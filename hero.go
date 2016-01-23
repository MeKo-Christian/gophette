package main

const (
	HeroAccelerationX = 2
	HeroDecelerationX = 1
	HeroMaxSpeedX     = 10

	HeroRunFrameDelay = 4
)

type Hero struct {
	Direction int
	X, Y      int
	SpeedX    int
	SpeedY    int

	runFrames     [DirectionCount][3]Image
	jumpFrames    [DirectionCount]Image
	runFrameIndex int
	nextRunFrame  int
}

func NewHero(assets AssetLoader) *Hero {
	return &Hero{
		runFrames: [DirectionCount][3]Image{
			[3]Image{
				assets.LoadImage("gophette_left_run1"),
				assets.LoadImage("gophette_left_run2"),
				assets.LoadImage("gophette_left_run3"),
			},
			[3]Image{
				assets.LoadImage("gophette_right_run1"),
				assets.LoadImage("gophette_right_run2"),
				assets.LoadImage("gophette_right_run3"),
			},
		},
		jumpFrames: [DirectionCount]Image{
			assets.LoadImage("gophette_left_jump"),
			assets.LoadImage("gophette_right_jump"),
		},
	}
}

func (h *Hero) Render() {
	// the order of animation images is 0,1,0,2  0,1,0,2  0,1 ...
	frameIndex := h.runFrameIndex
	if frameIndex == 2 {
		frameIndex = 0
	}
	if frameIndex == 3 {
		frameIndex = 2
	}

	h.runFrames[h.Direction][frameIndex].DrawAt(h.X, h.Y)
}

func (h *Hero) Update() {
	if h.SpeedX < 0 {
		h.Direction = LeftDirectionIndex
	}
	if h.SpeedX > 0 {
		h.Direction = RightDirectionIndex
	}

	if h.SpeedX == 0 {
		h.runFrameIndex = 0
		h.nextRunFrame = 0
	} else {
		h.nextRunFrame--
		if h.nextRunFrame <= 0 {
			h.runFrameIndex = (h.runFrameIndex + 1) % 4
			h.nextRunFrame = HeroRunFrameDelay
		}
	}

	h.X += h.SpeedX
}
