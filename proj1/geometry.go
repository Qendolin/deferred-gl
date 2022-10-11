package main

import (
	"encoding/binary"
	"log"

	"github.com/go-gl/gl/v4.5-core/gl"
)

type buffer struct {
	glId      uint32
	size      int
	flags     uint32
	immutable bool
}

type UnboundBuffer interface {
	Id() uint32
	Allocate(data any, flags int)
	AllocateMutable(data any, usage int)
	AllocateEmpty(size int, flags int)
	AllocateEmptyMutable(size int, usage int)
	Grow(size int) bool
	Write(offset int, data any)
	WriteRange(offset int, size int, data any)
	WriteIndex(index int, data any)
	Size() int
	Bind(target uint32) BoundBuffer
}

type BoundBuffer interface {
	UnboundBuffer
}

func NewBuffer() UnboundBuffer {
	var id uint32
	gl.CreateBuffers(1, &id)
	return &buffer{
		glId: id,
	}
}

func (vbo *buffer) Id() uint32 {
	return vbo.glId
}

func (vbo *buffer) Bind(target uint32) BoundBuffer {
	GlState.BindBuffer(target, vbo.glId)
	return BoundBuffer(vbo)
}

func (vbo *buffer) Size() int {
	return vbo.size
}

func (vbo *buffer) AllocateEmpty(size int, flags int) {
	if vbo.immutable {
		log.Panicf("VBO is immutable")
	}
	gl.NamedBufferStorage(vbo.glId, size, nil, uint32(flags))
	vbo.size = size
	vbo.flags = uint32(flags)
	vbo.immutable = true
}

func (vbo *buffer) AllocateEmptyMutable(size int, usage int) {
	if vbo.immutable {
		log.Panicf("VBO is immutable")
	}
	gl.NamedBufferData(vbo.glId, size, nil, uint32(usage))
	vbo.flags = uint32(usage)
	vbo.size = size
}

func (vbo *buffer) Allocate(data any, flags int) {
	if vbo.immutable {
		log.Panicf("VBO is immutable")
	}
	size := binary.Size(data)
	if size == -1 {
		log.Panicf("%v does not have a fixed size", data)
	}
	gl.NamedBufferStorage(vbo.glId, size, Pointer(data), uint32(flags))
	vbo.size = size
	vbo.flags = uint32(flags)
	vbo.immutable = true
}

func (vbo *buffer) AllocateMutable(data any, usage int) {
	if vbo.immutable {
		log.Panicf("VBO is immutable")
	}
	size := binary.Size(data)
	if size == -1 {
		log.Panicf("%v does not have a fixed size", data)
	}
	gl.NamedBufferData(vbo.glId, size, Pointer(data), uint32(usage))
	vbo.flags = uint32(usage)
	vbo.size = size
}

func (vbo *buffer) Grow(size int) bool {
	if size < vbo.size {
		return false
	}
	newSize := vbo.size
	doubleSize := newSize + newSize
	if size > doubleSize {
		newSize = size
	} else if vbo.size < 16_384 {
		// 16.384 is enough to hold 256 mat4
		newSize = doubleSize
	} else {
		// Check 0 < newcap to detect overflow
		// and prevent an infinite loop.
		for 0 < newSize && newSize < size {
			// Grow 1.25x
			newSize += newSize / 4
		}
		// Set newcap to the requested cap when
		// the newcap calculation overflowed.
		if newSize <= 0 {
			newSize = size
		}
	}

	if vbo.immutable {
		var newBufferId uint32
		gl.CreateBuffers(1, &newBufferId)
		gl.NamedBufferStorage(newBufferId, newSize, nil, vbo.flags)
		gl.CopyNamedBufferSubData(vbo.glId, newBufferId, 0, 0, vbo.size)
		gl.DeleteBuffers(1, &vbo.glId)
		vbo.glId = newBufferId
	} else {
		var copyBufferId uint32
		gl.CreateBuffers(1, &copyBufferId)
		gl.NamedBufferStorage(copyBufferId, vbo.size, nil, 0)
		gl.CopyNamedBufferSubData(vbo.glId, copyBufferId, 0, 0, vbo.size)
		gl.NamedBufferData(vbo.glId, newSize, nil, vbo.flags)
		gl.CopyNamedBufferSubData(copyBufferId, vbo.glId, 0, 0, vbo.size)
		gl.DeleteBuffers(1, &copyBufferId)
	}
	vbo.size = newSize
	return true
}

func (vbo *buffer) Write(offset int, data any) {
	size := binary.Size(data)
	if size == -1 {
		log.Panicf("%T does not have a fixed size", data)
	}
	gl.NamedBufferSubData(vbo.glId, int(offset), size, Pointer(data))
}

func (vbo *buffer) WriteRange(offset int, size int, data any) {
	gl.NamedBufferSubData(vbo.glId, int(offset), size, Pointer(data))
}

func (vbo *buffer) WriteIndex(index int, data any) {
	size := binary.Size(data)
	if size == -1 {
		log.Panicf("%T does not have a fixed size", data)
	}
	gl.NamedBufferSubData(vbo.glId, int(index*size), size, Pointer(data))
}

type vertexArray struct {
	glId uint32
}

type UnboundVertexArray interface {
	Layout(bufferIndex int, attributeIndex int, size int, dataType int, normalized bool, offset int)
	LayoutI(bufferIndex int, attributeIndex int, size int, dataType int, offset int)
	BindBuffer(bufferIndex int, vbo UnboundBuffer, offset int, stride int)
	BindElementBuffer(ibo UnboundBuffer)
	AttribDivisor(bufferIndex, divisor int)
	Id() uint32
	Bind() BoundVertexArray
}

type BoundVertexArray interface {
	UnboundVertexArray
}

func NewVertexArray() UnboundVertexArray {
	var id uint32
	gl.CreateVertexArrays(1, &id)
	return &vertexArray{
		glId: id,
	}
}

func (vao *vertexArray) Bind() BoundVertexArray {
	GlState.BindVertexArray(vao.glId)
	return BoundVertexArray(vao)
}

func (vao *vertexArray) Id() uint32 {
	return vao.glId
}

func (vao *vertexArray) Layout(bufferIndex int, attributeIndex int, size int, dataType int, normalized bool, offset int) {
	gl.EnableVertexArrayAttrib(vao.glId, uint32(attributeIndex))
	gl.VertexArrayAttribFormat(vao.glId, uint32(attributeIndex), int32(size), uint32(dataType), normalized, uint32(offset))
	gl.VertexArrayAttribBinding(vao.glId, uint32(attributeIndex), uint32(bufferIndex))
}

func (vao *vertexArray) LayoutI(bufferIndex int, attributeIndex int, size int, dataType int, offset int) {
	gl.EnableVertexArrayAttrib(vao.glId, uint32(attributeIndex))
	gl.VertexArrayAttribIFormat(vao.glId, uint32(attributeIndex), int32(size), uint32(dataType), uint32(offset))
	gl.VertexArrayAttribBinding(vao.glId, uint32(attributeIndex), uint32(bufferIndex))
}

func (vao *vertexArray) BindBuffer(bufferIndex int, vbo UnboundBuffer, offset int, stride int) {
	gl.VertexArrayVertexBuffer(vao.glId, uint32(bufferIndex), vbo.Id(), offset, int32(stride))
}

func (vao *vertexArray) BindElementBuffer(ibo UnboundBuffer) {
	gl.VertexArrayElementBuffer(vao.glId, ibo.Id())
}

func (vao *vertexArray) AttribDivisor(bufferIndex, divisor int) {
	gl.VertexArrayBindingDivisor(vao.glId, uint32(bufferIndex), uint32(divisor))
}
