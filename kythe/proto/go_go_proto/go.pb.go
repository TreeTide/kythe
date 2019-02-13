// Code generated by protoc-gen-go. DO NOT EDIT.
// source: kythe/proto/go.proto

package go_go_proto

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type GoDetails struct {
	Goos                 string   `protobuf:"bytes,1,opt,name=goos,proto3" json:"goos,omitempty"`
	Goarch               string   `protobuf:"bytes,2,opt,name=goarch,proto3" json:"goarch,omitempty"`
	Goroot               string   `protobuf:"bytes,3,opt,name=goroot,proto3" json:"goroot,omitempty"`
	Gopath               string   `protobuf:"bytes,4,opt,name=gopath,proto3" json:"gopath,omitempty"`
	Compiler             string   `protobuf:"bytes,5,opt,name=compiler,proto3" json:"compiler,omitempty"`
	BuildTags            []string `protobuf:"bytes,6,rep,name=build_tags,json=buildTags,proto3" json:"build_tags,omitempty"`
	CgoEnabled           bool     `protobuf:"varint,7,opt,name=cgo_enabled,json=cgoEnabled,proto3" json:"cgo_enabled,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GoDetails) Reset()         { *m = GoDetails{} }
func (m *GoDetails) String() string { return proto.CompactTextString(m) }
func (*GoDetails) ProtoMessage()    {}
func (*GoDetails) Descriptor() ([]byte, []int) {
	return fileDescriptor_go_0825452901a7d082, []int{0}
}
func (m *GoDetails) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GoDetails.Unmarshal(m, b)
}
func (m *GoDetails) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GoDetails.Marshal(b, m, deterministic)
}
func (dst *GoDetails) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GoDetails.Merge(dst, src)
}
func (m *GoDetails) XXX_Size() int {
	return xxx_messageInfo_GoDetails.Size(m)
}
func (m *GoDetails) XXX_DiscardUnknown() {
	xxx_messageInfo_GoDetails.DiscardUnknown(m)
}

var xxx_messageInfo_GoDetails proto.InternalMessageInfo

func (m *GoDetails) GetGoos() string {
	if m != nil {
		return m.Goos
	}
	return ""
}

func (m *GoDetails) GetGoarch() string {
	if m != nil {
		return m.Goarch
	}
	return ""
}

func (m *GoDetails) GetGoroot() string {
	if m != nil {
		return m.Goroot
	}
	return ""
}

func (m *GoDetails) GetGopath() string {
	if m != nil {
		return m.Gopath
	}
	return ""
}

func (m *GoDetails) GetCompiler() string {
	if m != nil {
		return m.Compiler
	}
	return ""
}

func (m *GoDetails) GetBuildTags() []string {
	if m != nil {
		return m.BuildTags
	}
	return nil
}

func (m *GoDetails) GetCgoEnabled() bool {
	if m != nil {
		return m.CgoEnabled
	}
	return false
}

type GoPackageInfo struct {
	ImportPath           string   `protobuf:"bytes,1,opt,name=import_path,json=importPath,proto3" json:"import_path,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GoPackageInfo) Reset()         { *m = GoPackageInfo{} }
func (m *GoPackageInfo) String() string { return proto.CompactTextString(m) }
func (*GoPackageInfo) ProtoMessage()    {}
func (*GoPackageInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_go_0825452901a7d082, []int{1}
}
func (m *GoPackageInfo) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GoPackageInfo.Unmarshal(m, b)
}
func (m *GoPackageInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GoPackageInfo.Marshal(b, m, deterministic)
}
func (dst *GoPackageInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GoPackageInfo.Merge(dst, src)
}
func (m *GoPackageInfo) XXX_Size() int {
	return xxx_messageInfo_GoPackageInfo.Size(m)
}
func (m *GoPackageInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_GoPackageInfo.DiscardUnknown(m)
}

var xxx_messageInfo_GoPackageInfo proto.InternalMessageInfo

func (m *GoPackageInfo) GetImportPath() string {
	if m != nil {
		return m.ImportPath
	}
	return ""
}

func init() {
	proto.RegisterType((*GoDetails)(nil), "kythe.proto.GoDetails")
	proto.RegisterType((*GoPackageInfo)(nil), "kythe.proto.GoPackageInfo")
}

func init() { proto.RegisterFile("kythe/proto/go.proto", fileDescriptor_go_0825452901a7d082) }

var fileDescriptor_go_0825452901a7d082 = []byte{
	// 244 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x4c, 0xd0, 0x41, 0x4b, 0xfb, 0x30,
	0x18, 0xc7, 0x71, 0xfa, 0xdf, 0xfe, 0x75, 0x7d, 0x86, 0x97, 0x20, 0x12, 0x04, 0x59, 0xdd, 0xa9,
	0xa7, 0x4e, 0xf0, 0x1d, 0x88, 0x32, 0xbc, 0x8d, 0xe2, 0xbd, 0xa4, 0x69, 0x7c, 0x5a, 0x96, 0xee,
	0x57, 0x9a, 0x28, 0xf8, 0xfa, 0x7c, 0x63, 0xd2, 0xc4, 0x89, 0xa7, 0x3c, 0xdf, 0x4f, 0x08, 0x24,
	0xa1, 0xab, 0xe3, 0xa7, 0xef, 0xcc, 0x6e, 0x9c, 0xe0, 0xb1, 0x63, 0x94, 0x61, 0x10, 0xeb, 0xa0,
	0x31, 0xb6, 0x5f, 0x09, 0x65, 0x7b, 0x3c, 0x19, 0xaf, 0x7a, 0xeb, 0x84, 0xa0, 0x25, 0x03, 0x4e,
	0x26, 0x79, 0x52, 0x64, 0x55, 0x98, 0xc5, 0x35, 0xa5, 0x0c, 0x35, 0xe9, 0x4e, 0xfe, 0x0b, 0xfa,
	0x53, 0xd1, 0x27, 0xc0, 0xcb, 0xc5, 0xd9, 0xe7, 0x8a, 0x3e, 0x2a, 0xdf, 0xc9, 0xe5, 0xd9, 0xe7,
	0x12, 0x37, 0xb4, 0xd2, 0x18, 0xc6, 0xde, 0x9a, 0x49, 0xfe, 0x0f, 0x3b, 0xbf, 0x2d, 0x6e, 0x89,
	0x9a, 0xf7, 0xde, 0xb6, 0xb5, 0x57, 0xec, 0x64, 0x9a, 0x2f, 0x8a, 0xac, 0xca, 0x82, 0xbc, 0x2a,
	0x76, 0x62, 0x43, 0x6b, 0xcd, 0xa8, 0xcd, 0x49, 0x35, 0xd6, 0xb4, 0xf2, 0x22, 0x4f, 0x8a, 0x55,
	0x45, 0x9a, 0xf1, 0x1c, 0x65, 0x7b, 0x4f, 0x97, 0x7b, 0x1c, 0x94, 0x3e, 0x2a, 0x36, 0x2f, 0xa7,
	0x37, 0xcc, 0x27, 0xfa, 0x61, 0xc4, 0xe4, 0xeb, 0x70, 0x93, 0xf8, 0x1e, 0x8a, 0x74, 0x50, 0xbe,
	0x7b, 0xbc, 0xa3, 0x8d, 0xc6, 0x50, 0x32, 0xc0, 0xd6, 0x94, 0xad, 0xf9, 0xf0, 0x80, 0x75, 0xe5,
	0x9f, 0xaf, 0x69, 0xd2, 0xb0, 0x3c, 0x7c, 0x07, 0x00, 0x00, 0xff, 0xff, 0x90, 0xf7, 0x30, 0xdb,
	0x46, 0x01, 0x00, 0x00,
}
