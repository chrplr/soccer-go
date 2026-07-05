package main

import (
	"embed"
	"math/rand"
	"path"
	"runtime"
	"strconv"

	"github.com/Zyko0/go-sdl3/mixer"
	"github.com/Zyko0/go-sdl3/sdl"
)

// audioFS embeds the sound effects and music into the binary.
//
//go:embed sounds music
var audioFS embed.FS

// Audio wraps SDL3_mixer. Every operation is best-effort: if the mixer cannot be
// created (no audio device) all methods become no-ops, matching the original
// game's silent handling of sound errors.
type Audio struct {
	mixer  *mixer.Mixer
	sounds map[string]*mixer.Audio
	music  *mixer.Track // looping theme (menu)
	crowd  *mixer.Track // looping crowd (in-game)
}

func NewAudio() *Audio {
	a := &Audio{sounds: make(map[string]*mixer.Audio)}

	// SDL3_mixer has no js/wasm bindings yet: in the browser, skip audio init
	// entirely and run silently (every play method no-ops on a nil mixer).
	if runtime.GOOS == "js" {
		return a
	}

	if err := mixer.Init(); err != nil {
		return a
	}
	m, err := mixer.CreateMixerDevice(sdl.AUDIO_DEVICE_DEFAULT_PLAYBACK, nil)
	if err != nil {
		return a
	}
	a.mixer = m

	entries, _ := audioFS.ReadDir("sounds")
	for _, e := range entries {
		fname := e.Name()
		if path.Ext(fname) != ".ogg" {
			continue
		}
		if snd := loadAudioFromFS(m, "sounds/"+fname); snd != nil {
			a.sounds[fname[:len(fname)-len(".ogg")]] = snd
		}
	}

	a.music = a.loopingTrack(m, "music/theme.ogg", 0.5)
	// The crowd loop is one of the sound effects rather than music.
	if snd, ok := a.sounds["crowd"]; ok {
		if t, err := m.CreateTrack(); err == nil {
			t.SetAudio(snd)
			t.SetLoops(-1)
			a.crowd = t
		}
	}
	return a
}

// loadAudioFromFS decodes an embedded audio file into an in-memory Audio via an
// SDL IOStream (predecoded, so no stream stays open afterwards).
func loadAudioFromFS(m *mixer.Mixer, p string) *mixer.Audio {
	data, err := audioFS.ReadFile(p)
	if err != nil {
		return nil
	}
	stream, err := sdl.IOFromConstMem(data)
	if err != nil {
		return nil
	}
	snd, err := m.LoadAudio_IO(stream, true, true) // predecode + closeio
	if err != nil {
		return nil
	}
	return snd
}

func (a *Audio) loopingTrack(m *mixer.Mixer, p string, gain float32) *mixer.Track {
	audio := loadAudioFromFS(m, p)
	if audio == nil {
		return nil
	}
	t, err := m.CreateTrack()
	if err != nil {
		return nil
	}
	t.SetAudio(audio)
	t.SetLoops(-1)
	t.SetGain(gain)
	return t
}

// PlaySound plays one of a family of variants: <name>0 .. <name>(count-1).
func (a *Audio) PlaySound(name string, count int) {
	a.play(name + strconv.Itoa(rand.Intn(count)))
}

// Play plays a single sound by exact name (e.g. "start", "move").
func (a *Audio) Play(name string) { a.play(name) }

func (a *Audio) play(key string) {
	if a.mixer == nil {
		return
	}
	if snd, ok := a.sounds[key]; ok {
		a.mixer.PlayAudio(snd)
	}
}

// StartMenuMusic plays the looping theme and stops the crowd.
func (a *Audio) StartMenuMusic() {
	if a.music != nil {
		a.music.Play(0)
	}
	if a.crowd != nil {
		a.crowd.Stop(0)
	}
}

// StartMatchAudio fades out the theme, starts the crowd loop and plays the whistle.
func (a *Audio) StartMatchAudio() {
	if a.music != nil {
		a.music.Stop(0)
	}
	if a.crowd != nil {
		a.crowd.Play(0)
	}
	a.Play("start")
}

func (a *Audio) Destroy() {
	if a.mixer != nil {
		a.mixer.Destroy()
	}
}
