package main

import (
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"math"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type texture struct {
	glId       uint32
	dimensions uint32
	width      int32
	height     int32
	depth      int32
}

type UnboundTexture interface {
	Id() uint32
	Bind(unit int) BoundTexture
	Allocate(levels int, internalFormat uint32, width, height, depth int)
	Load(level int, width, height, depth int, format uint32, data any)
	MipmapLevels(base, max int)
	DepthStencilTextureMode(mode int32)
	CreateView(dimensions, internalFormat uint32, minLevel, maxLevel, minLayer, maxLayer int) UnboundTexture
	GenerateMipmap()
}

type BoundTexture interface {
	UnboundTexture
}

func NewTexture(dimensions uint32) UnboundTexture {
	var id uint32
	gl.CreateTextures(dimensions, 1, &id)
	if GlEnv.UseIntelTextureBindingFix {
		GlEnv.IntelTextureBindingTargets[id] = dimensions
	}
	return &texture{
		glId:       id,
		dimensions: dimensions,
	}
}

func (tex *texture) Dimensions() int {
	switch tex.dimensions {
	case gl.TEXTURE_1D, gl.TEXTURE_BUFFER:
		return 1
	case gl.TEXTURE_3D, gl.TEXTURE_2D_ARRAY, gl.TEXTURE_CUBE_MAP:
		return 3
	case gl.TEXTURE_2D, gl.TEXTURE_1D_ARRAY, gl.TEXTURE_CUBE_MAP_NEGATIVE_X, gl.TEXTURE_CUBE_MAP_NEGATIVE_Y, gl.TEXTURE_CUBE_MAP_NEGATIVE_Z,
		gl.TEXTURE_CUBE_MAP_POSITIVE_X, gl.TEXTURE_CUBE_MAP_POSITIVE_Y, gl.TEXTURE_CUBE_MAP_POSITIVE_Z:
		return 2
	default:
		return 0
	}
}

func (tex *texture) Id() uint32 {
	return tex.glId
}

func (tex *texture) Bind(unit int) BoundTexture {
	GlState.BindTextureUnit(unit, tex.glId)
	return BoundTexture(tex)
}

func (tex *texture) CreateView(dimensions, internalFormat uint32, minLevel, maxLevel, minLayer, maxLayer int) UnboundTexture {
	var viewId uint32
	gl.GenTextures(1, &viewId)
	gl.TextureView(viewId, dimensions, tex.glId, internalFormat, uint32(minLevel), uint32(maxLevel-minLevel+1), uint32(minLayer), uint32(maxLayer-minLayer+1))
	if GlEnv.UseIntelTextureBindingFix {
		GlEnv.IntelTextureBindingTargets[viewId] = dimensions
	}
	return &texture{
		glId:       viewId,
		dimensions: dimensions,
	}
}

func (tex *texture) Allocate(levels int, internalFormat uint32, width, height, depth int) {
	if levels == 0 {
		max := math.Max(math.Max(float64(width), float64(height)), float64(depth))
		levels = int(math.Log2(max))
		if levels == 0 {
			levels = 1
		}
	}
	tex.width = int32(width)
	tex.height = int32(height)
	tex.depth = int32(depth)
	switch tex.Dimensions() {
	case 1:
		gl.TextureStorage1D(tex.glId, int32(levels), internalFormat, int32(width))
	case 2:
		gl.TextureStorage2D(tex.glId, int32(levels), internalFormat, int32(width), int32(height))
	case 3:
		gl.TextureStorage3D(tex.glId, int32(levels), internalFormat, int32(width), int32(height), int32(depth))
	}
}

func (tex *texture) Load(level int, width, height, depth int, format uint32, data any) {
	dataType, _ := getGlType(data)
	switch tex.Dimensions() {
	case 1:
		gl.TextureSubImage1D(tex.glId, int32(level), 0, int32(width), format, dataType, Pointer(data))
	case 2:
		gl.TextureSubImage2D(tex.glId, int32(level), 0, 0, int32(width), int32(height), format, dataType, Pointer(data))
	case 3:
		gl.TextureSubImage3D(tex.glId, int32(level), 0, 0, 0, int32(width), int32(height), int32(depth), format, dataType, Pointer(data))
	}
}

func (tex *texture) GenerateMipmap() {
	gl.GenerateTextureMipmap(tex.glId)
}

func (tex *texture) MipmapLevels(base, max int) {
	gl.TextureParameteri(tex.glId, gl.TEXTURE_BASE_LEVEL, int32(base))
	gl.TextureParameteri(tex.glId, gl.TEXTURE_MAX_LEVEL, int32(max))
}

func (tex *texture) DepthStencilTextureMode(mode int32) {
	gl.TextureParameteri(tex.glId, gl.DEPTH_STENCIL_TEXTURE_MODE, mode)
}

func getGlType(data any) (glType uint32, float bool) {
	switch data.(type) {
	case byte, []byte, *byte:
		return gl.UNSIGNED_BYTE, false
	case int8, []int8, *int8:
		return gl.BYTE, false
	case int16, []int16, *int16:
		return gl.SHORT, false
	case uint16, []uint16, *uint16:
		return gl.UNSIGNED_SHORT, false
	case int32, []int32, *int32:
		return gl.INT, false
	case uint32, []uint32, *uint32:
		return gl.UNSIGNED_INT, false
	case float32, []float32, *float32, mgl32.Vec2, []mgl32.Vec2, mgl32.Vec3, []mgl32.Vec3, mgl32.Vec4, []mgl32.Vec4:
		return gl.FLOAT, true
	case float64, []float64, *float64:
		return gl.DOUBLE, true
	}
	log.Panicf("invalid type: %T", data)
	return 0, false
}

type sampler struct {
	glId uint32
}

type UnboundSampler interface {
	Id() uint32
	Bind(unit int) BoundSampler
	FilterMode(min, mag int32)
	WrapMode(s, t, r int32)
	CompareMode(mode, fn int32)
	BorderColor(color mgl32.Vec4)
}

type BoundSampler interface {
	UnboundSampler
}

func NewSampler() UnboundSampler {
	var id uint32
	gl.CreateSamplers(1, &id)
	return &sampler{
		glId: id,
	}
}

func (s *sampler) Id() uint32 {
	return s.glId
}

func (s *sampler) Bind(unit int) BoundSampler {
	GlState.BindSampler(unit, s.glId)
	return BoundSampler(s)
}

func (s *sampler) FilterMode(min, mag int32) {
	if min != 0 {
		gl.SamplerParameteri(s.glId, gl.TEXTURE_MIN_FILTER, min)
	}
	if mag != 0 {
		gl.SamplerParameteri(s.glId, gl.TEXTURE_MIN_FILTER, mag)
	}
}

func (sampler *sampler) WrapMode(s, t, r int32) {
	if s != 0 {
		gl.SamplerParameteri(sampler.glId, gl.TEXTURE_WRAP_S, s)
	}
	if t != 0 {
		gl.SamplerParameteri(sampler.glId, gl.TEXTURE_WRAP_T, t)
	}
	if r != 0 {
		gl.SamplerParameteri(sampler.glId, gl.TEXTURE_WRAP_R, r)
	}
}

func (sampler *sampler) CompareMode(mode, fn int32) {
	gl.SamplerParameteri(sampler.glId, gl.TEXTURE_COMPARE_MODE, mode)
	if fn != 0 {
		gl.SamplerParameteri(sampler.glId, gl.TEXTURE_COMPARE_FUNC, mode)
	}
}

func (sampler *sampler) BorderColor(color mgl32.Vec4) {
	gl.SamplerParameterfv(sampler.glId, gl.TEXTURE_BORDER_COLOR, &color[0])
}

// TODO: Rename
func LoadImage(r io.Reader) *image.RGBA {
	img, _, err := image.Decode(r)
	if err != nil {
		log.Panic(err)
	}
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, img.Bounds(), img, image.Point{}, draw.Src)
	return rgba
}
