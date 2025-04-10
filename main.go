//go:build !windows || sdl2
// +build !windows sdl2

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"runtime"
	"time"

	"github.com/gonutz/blob"
	"github.com/gonutz/payload"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/mix"
	"github.com/veandco/go-sdl2/sdl"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	sdl.SetHint(sdl.HINT_RENDER_VSYNC, "1")

	check(sdl.Init(sdl.INIT_EVERYTHING))
	defer sdl.Quit()

	check(mix.Init(mix.INIT_OGG))
	defer mix.Quit()
	check(mix.OpenAudio(44100, mix.DEFAULT_FORMAT, 1, 512))
	defer mix.CloseAudio()

	if err := img.Init(img.INIT_PNG); err != nil {
		panic(fmt.Errorf("error initializing SDL_image: %v", err))
	}
	defer img.Quit()

	window, renderer, err := sdl.CreateWindowAndRenderer(
		640, 480,
		sdl.WINDOW_RESIZABLE,
	)
	check(err)
	defer renderer.Destroy()
	defer window.Destroy()
	window.SetTitle("Gophette's Adventures")
	window.SetSize(800, 600)
	sdl.ShowCursor(0)

	window.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
	fullscreen := true

	w, h := window.GetSize()
	camera := newWindowCamera(int(w), int(h))

	assetLoader := newSDLAssetLoader(camera, renderer)
	defer assetLoader.close()

	// charIndex selects which character is being controlled by the user, for
	// the final game this must be 0 but for creating the "AI" for Barney, set
	// this to 1 and delete the recorded inputs so they are not applied
	// additionally to the user controls

	var charIndex int
	const recordingAI = false // NOTE switch for development mode
	if !recordingAI {
		charIndex = 0
	} else {
		charIndex = 1
		recordedInputs = recordedInputs[:0]
		recordingInput = true
	}

	game := NewGame(
		assetLoader,
		&sdlGraphics{renderer, camera},
		camera,
		charIndex,
	)

	musicData, found := assetLoader.resources.GetByID("music")
	if found {
		musicRWOps, err := sdl.RWFromMem(musicData)
		check(err)

		music, err := mix.LoadMUSRW(musicRWOps, 1)
		if err != nil {
			fmt.Println("error loading music:", err)
		} else {
			defer music.Free()
			music.FadeIn(-1, 500)
		}
	}

	frameTime := time.Second / 65
	lastUpdate := time.Now().Add(-frameTime)

	for game.Running() {
		for e := sdl.PollEvent(); e != nil; e = sdl.PollEvent() {
			switch event := e.(type) {
			case *sdl.KeyboardEvent:
				if event.Type == sdl.KEYDOWN {
					if event.Repeat == 0 {
						switch event.Keysym.Sym {
						case sdl.K_LEFT:
							game.HandleInput(InputEvent{GoLeft, true, charIndex})
						case sdl.K_RIGHT:
							game.HandleInput(InputEvent{GoRight, true, charIndex})
						case sdl.K_UP, sdl.K_SPACE, sdl.K_LCTRL:
							game.HandleInput(InputEvent{Jump, true, charIndex})
						case sdl.K_ESCAPE:
							game.HandleInput(InputEvent{QuitGame, true, charIndex})
						}
					}
				}
				if event.Type == sdl.KEYUP {
					switch event.Keysym.Sym {
					case sdl.K_LEFT:
						game.HandleInput(InputEvent{GoLeft, false, charIndex})
					case sdl.K_RIGHT:
						game.HandleInput(InputEvent{GoRight, false, charIndex})
					case sdl.K_UP, sdl.K_SPACE, sdl.K_LCTRL:
						game.HandleInput(InputEvent{Jump, false, charIndex})
					case sdl.K_F11:
						if fullscreen {
							window.SetFullscreen(0)
						} else {
							window.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
						}
						fullscreen = !fullscreen
					case sdl.K_ESCAPE:
						game.HandleInput(InputEvent{QuitGame, false, charIndex})
					}
				}

			case *sdl.WindowEvent:
				if event.Event == sdl.WINDOWEVENT_SIZE_CHANGED {
					width, height := int(event.Data1), int(event.Data2)
					camera.setWindowSize(width, height)
				}
			case *sdl.QuitEvent:
				game.HandleInput(InputEvent{QuitGame, true, charIndex})
			}
		}

		now := time.Now()
		dt := now.Sub(lastUpdate)
		if dt > frameTime {
			game.Update()
			lastUpdate = now
		}

		check(renderer.SetDrawColor(0, 95, 83, 255))
		check(renderer.Clear())
		game.Render()
		renderer.Present()
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type textureImage struct {
	renderer *sdl.Renderer
	camera   *windowCamera
	texture  *sdl.Texture
	source   sdl.Rect
}

func (img *textureImage) DrawAt(x, y int) {
	dx, dy := img.camera.offset()
	dest := sdl.Rect{int32(x + dx), int32(y + dy), img.source.W, img.source.H}
	check(img.renderer.Copy(img.texture, &img.source, &dest))
}

func (img *textureImage) Size() (int, int) {
	return int(img.source.W), int(img.source.H)
}

type wavSound struct {
	chunk *mix.Chunk
}

func (s *wavSound) PlayOnce() {
	s.chunk.Play(-1, 0)
}

func (s *wavSound) Length() time.Duration {
	return time.Millisecond * time.Duration(s.chunk.LengthInMs())
}

type sdlAssetLoader struct {
	resources    *blob.Blob
	camera       *windowCamera
	renderer     *sdl.Renderer
	textureAtlas *sdl.Texture
	images       map[string]*textureImage
	sounds       map[string]*wavSound
}

func (l *sdlAssetLoader) loadResources() error {
	resourceData, err := payload.Read()
	if err != nil {
		return err
	}
	l.resources, err = blob.Read(bytes.NewBuffer(resourceData))
	if err != nil {
		return err
	}

	// load the texture atlas
	atlas, found := l.resources.GetByID("atlas")
	if !found {
		panic("texture atlas not found in resources")
	}
	rwOps, err := sdl.RWFromMem(atlas)
	surface, err := img.LoadRW(rwOps, false)
	check(err)
	defer surface.Free()
	texture, err := l.renderer.CreateTextureFromSurface(surface)
	check(err)
	l.textureAtlas = texture

	return err
}

func newSDLAssetLoader(cam *windowCamera, renderer *sdl.Renderer) *sdlAssetLoader {
	l := &sdlAssetLoader{
		camera:   cam,
		renderer: renderer,
		images:   make(map[string]*textureImage),
		sounds:   make(map[string]*wavSound),
	}
	check(l.loadResources())
	return l
}

func (l *sdlAssetLoader) LoadImage(id string) Image {
	if img, ok := l.images[id]; ok {
		return img
	}
	data, _ := l.resources.GetByID(id)
	if data == nil {
		panic("unknown image resource: " + id)
	}

	// the loaded data is a binary rectangle that describes the location in
	// the texture atlas for the given image ID
	var bounds rect
	check(binary.Read(bytes.NewReader(data), binary.LittleEndian, &bounds))

	image := &textureImage{
		l.renderer,
		l.camera,
		l.textureAtlas,
		sdl.Rect{bounds.X, bounds.Y, bounds.W, bounds.H},
	}
	l.images[id] = image

	return image
}

func (l *sdlAssetLoader) LoadSound(id string) Sound {
	if sound, ok := l.sounds[id]; ok {
		return sound
	}
	data, _ := l.resources.GetByID(id)
	if data == nil {
		panic("unknown sound resource: " + id)
	}

	rwOps, err := sdl.RWFromMem(data)
	check(err)
	chunk, err := mix.LoadWAVRW(rwOps, false)
	check(err)
	sound := &wavSound{chunk}
	l.sounds[id] = sound

	return sound
}

func (l *sdlAssetLoader) LoadRectangle(id string) Rectangle {
	data, found := l.resources.GetByID(id)
	if !found {
		panic("unknown rectangle resource: " + id)
	}
	reader := bytes.NewReader(data)
	var r rect
	check(binary.Read(reader, binary.LittleEndian, &r))
	return Rectangle{int(r.X), int(r.Y), int(r.W), int(r.H)}
}

type rect struct {
	X, Y, W, H int32
}

func (l *sdlAssetLoader) close() {
	for _, image := range l.images {
		image.texture.Destroy()
	}
	for _, sound := range l.sounds {
		sound.chunk.Free()
	}
}

type sdlGraphics struct {
	renderer *sdl.Renderer
	camera   *windowCamera
}

func (graphics *sdlGraphics) ClearScreen(r, g, b uint8) {
	check(graphics.renderer.SetDrawColor(r, g, b, 255))
	graphics.renderer.Clear()
}
