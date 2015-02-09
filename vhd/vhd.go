package vhd

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

const VHD_COOKIE = "636f6e6563746978"     // conectix
const VHD_DYN_COOKIE = "6378737061727365" // cxsparse
const VHD_CREATOR_APP = "676f2d766864"    // go-vhd
const VHD_CREATOR_HOST_OS = "5769326B"    // Win2k
const VHD_BLOCK_SIZE = 2 * 1024 * 1024    // 2MB
const VHD_HEADER_SIZE = 512
const SECTOR_SIZE = 512
const VHD_EXTRA_HEADER_SIZE = 1024

func fmtField(name, value string) {
	fmt.Printf("%-25s%s\n", name+":", value)
}

// https://groups.google.com/forum/#!msg/golang-nuts/d0nF_k4dSx4/rPGgfXv6QCoJ
func uuidgen() string {
	b := uuidgenBytes()
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func uuidgenBytes() []byte {
	f, err := os.Open("/dev/urandom")
	check(err)
	b := make([]byte, 16)
	f.Read(b)
	return b
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func hexs(a []byte) string {
	return "0x" + hex.EncodeToString(a[:])
}

func uuid(a []byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%04x",
		a[:4],
		a[4:6],
		a[6:8],
		a[8:10],
		a[10:16])
}

func calculateCHS(ts uint64) []uint {
	var sectorsPerTrack,
		heads,
		cylinderTimesHeads,
		cylinders float64
	totalSectors := float64(ts)

	ret := make([]uint, 3)

	if totalSectors > 65535*16*255 {
		totalSectors = 65535 * 16 * 255
	}

	if totalSectors >= 65535*16*63 {
		sectorsPerTrack = 255
		heads = 16
		cylinderTimesHeads = math.Floor(totalSectors / sectorsPerTrack)
	} else {
		sectorsPerTrack = 17
		cylinderTimesHeads = math.Floor(totalSectors / sectorsPerTrack)
		heads = math.Floor((cylinderTimesHeads + 1023) / 1024)
		if heads < 4 {
			heads = 4
		}
		if (cylinderTimesHeads >= (heads * 1024)) || heads > 16 {
			sectorsPerTrack = 31
			heads = 16
			cylinderTimesHeads = math.Floor(totalSectors / sectorsPerTrack)
		}
		if cylinderTimesHeads >= (heads * 1024) {
			sectorsPerTrack = 63
			heads = 16
			cylinderTimesHeads = math.Floor(totalSectors / sectorsPerTrack)
		}
	}

	cylinders = cylinderTimesHeads / heads

	// This will floor the values
	ret[0] = uint(cylinders)
	ret[1] = uint(heads)
	ret[2] = uint(sectorsPerTrack)

	return ret
}

/*
	utf16BytesToString converts UTF-16 encoded bytes, in big or
 	little endian byte order, to a UTF-8 encoded string.
 	http://stackoverflow.com/a/15794113
*/
func utf16BytesToString(b []byte, o binary.ByteOrder) string {
	utf := make([]uint16, (len(b)+(2-1))/2)
	for i := 0; i+(2-1) < len(b); i += 2 {
		utf[i/2] = o.Uint16(b[i:])
	}
	if len(b)/2 < len(utf) {
		utf[len(utf)-1] = utf8.RuneError
	}
	return string(utf16.Decode(utf))
}

/* VHD Dynamic and Differential Header */
/*
	Cookie 8
	Data Offset 8
	Table Offset 8
	Header Version 4
	Max Table Entries 4
	Block Size 4
	Checksum 4
	Parent Unique ID 16
	Parent Time Stamp 4
	Reserved 4
	Parent Unicode Name 512
	Parent Locator Entry 1 24
	Parent Locator Entry 2 24
	Parent Locator Entry 3 24
	Parent Locator Entry 4 24
	Parent Locator Entry 5 24
	Parent Locator Entry 6 24
	Parent Locator Entry 7 24
	Parent Locator Entry 8 24
	Reserved 256
*/
type VHDExtraHeader struct {
	Cookie              [8]byte
	DataOffset          [8]byte
	TableOffset         [8]byte
	HeaderVersion       [4]byte
	MaxTableEntries     [4]byte
	BlockSize           [4]byte
	Checksum            [4]byte
	ParentUUID          [16]byte
	ParentTimestamp     [4]byte
	Reserved            [4]byte
	ParentUnicodeName   [512]byte
	ParentLocatorEntry1 [24]byte
	ParentLocatorEntry2 [24]byte
	ParentLocatorEntry3 [24]byte
	ParentLocatorEntry4 [24]byte
	ParentLocatorEntry5 [24]byte
	ParentLocatorEntry6 [24]byte
	ParentLocatorEntry7 [24]byte
	ParentLocatorEntry8 [24]byte
	Reserved2           [256]byte
}

func (h *VHDExtraHeader) addChecksum() {
	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, h)
	checksum := 0
	bb := buffer.Bytes()

	for counter := 0; counter < VHD_EXTRA_HEADER_SIZE; counter++ {
		checksum += int(bb[counter])
	}

	binary.BigEndian.PutUint32(h.Checksum[:], uint32(^checksum))
}

func (header *VHDExtraHeader) CookieString() string {
	return string(header.Cookie[:])
}

/* VHD Header */
/*
 Cookie 8
 Features 4
 File Format Version 4
 Data Offset 8
 Time Stamp 4
 Creator Application 4
 Creator Version 4
 Creator Host OS 4
 Original Size 8
 Current Size 8
 Disk Geometry 4
 Disk Type 4
 Checksum 4
 Unique Id 16
 Saved State 1
 Reserved 427
*/
type VHDHeader struct {
	Cookie             [8]byte
	Features           [4]byte
	FileFormatVersion  [4]byte
	DataOffset         [8]byte
	Timestamp          [4]byte
	CreatorApplication [4]byte
	CreatorVersion     [4]byte
	CreatorHostOS      [4]byte
	OriginalSize       [8]byte
	CurrentSize        [8]byte
	DiskGeometry       [4]byte
	DiskType           [4]byte
	Checksum           [4]byte
	UniqueId           [16]byte
	SavedState         [1]byte
	Reserved           [427]byte
}

func (h *VHDHeader) addChecksum() {
	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, h)
	checksum := 0
	bb := buffer.Bytes()

	for counter := 0; counter < VHD_HEADER_SIZE; counter++ {
		checksum += int(bb[counter])
	}

	binary.BigEndian.PutUint32(h.Checksum[:], uint32(^checksum))
}

func (h *VHDHeader) DiskTypeStr() (dt string) {
	switch h.DiskType[3] {
	case 0x00:
		dt = "None"
	case 0x01:
		dt = "Deprecated"
	case 0x02:
		dt = "Fixed"
	case 0x03:
		dt = "Dynamic"
	case 0x04:
		dt = "Differential"
	case 0x05:
		dt = "Reserved"
	case 0x06:
		dt = "Reserved"
	default:
		panic("Invalid disk type detected!")
	}

	return
}

func (h *VHDHeader) TimestampTime() time.Time {
	tstamp := binary.BigEndian.Uint32(h.Timestamp[:])
	return time.Unix(int64(946684800+tstamp), 0)
}

func readVHDExtraHeader(f *os.File) {
	vhdHeader := make([]byte, 1024)
	_, err := f.Read(vhdHeader)
	check(err)

	var header VHDExtraHeader
	binary.Read(bytes.NewBuffer(vhdHeader[:]), binary.BigEndian, &header)

	fmtField("Cookie", fmt.Sprintf("%s (%s)",
		hexs(header.Cookie[:]), header.CookieString()))
	fmtField("Data offset", hexs(header.DataOffset[:]))
	fmtField("Table offset", hexs(header.TableOffset[:]))
	fmtField("Header version", hexs(header.HeaderVersion[:]))
	fmtField("Max table entries", hexs(header.MaxTableEntries[:]))
	fmtField("Block size", hexs(header.BlockSize[:]))
	fmtField("Checksum", hexs(header.Checksum[:]))
	fmtField("Parent UUID", uuid(header.ParentUUID[:]))

	// Seconds since January 1, 1970 12:00:00 AM in UTC/GMT.
	// 946684800 = January 1, 2000 12:00:00 AM in UTC/GMT.
	tstamp := binary.BigEndian.Uint32(header.ParentTimestamp[:])
	t := time.Unix(int64(946684800+tstamp), 0)
	fmtField("Parent timestamp", fmt.Sprintf("%s", t))

	fmtField("Reserved", hexs(header.Reserved[:]))
	parentName := utf16BytesToString(header.ParentUnicodeName[:],
		binary.BigEndian)
	fmtField("Parent Name", parentName)
	// Parent locator entries ignored since it's a dynamic disk
	sum := 0
	for _, b := range header.Reserved2 {
		sum += int(b)
	}
	fmtField("Reserved2", strconv.Itoa(sum))
}

func readVHDHeader(vhdHeader []byte) VHDHeader {

	var header VHDHeader
	binary.Read(bytes.NewBuffer(vhdHeader[:]), binary.BigEndian, &header)

	//fmtField("Cookie", string(header.Cookie[:]))
	fmtField("Cookie", fmt.Sprintf("%s (%s)",
		hexs(header.Cookie[:]), string(header.Cookie[:])))
	fmtField("Features", hexs(header.Features[:]))
	fmtField("File format version", hexs(header.FileFormatVersion[:]))

	dataOffset := binary.BigEndian.Uint64(header.DataOffset[:])
	fmtField("Data offset",
		fmt.Sprintf("%s (%d bytes)", hexs(header.DataOffset[:]), dataOffset))

	//// Seconds since January 1, 1970 12:00:00 AM in UTC/GMT.
	//// 946684800 = January 1, 2000 12:00:00 AM in UTC/GMT.
	t := time.Unix(int64(946684800+binary.BigEndian.Uint32(header.Timestamp[:])), 0)
	fmtField("Timestamp", fmt.Sprintf("%s", t))

	fmtField("Creator application", string(header.CreatorApplication[:]))
	fmtField("Creator version", hexs(header.CreatorVersion[:]))
	fmtField("Creator OS", string(header.CreatorHostOS[:]))

	originalSize := binary.BigEndian.Uint64(header.OriginalSize[:])
	fmtField("Original size",
		fmt.Sprintf("%s ( %d bytes )", hexs(header.OriginalSize[:]), originalSize))

	currentSize := binary.BigEndian.Uint64(header.OriginalSize[:])
	fmtField("Current size",
		fmt.Sprintf("%s ( %d bytes )", hexs(header.CurrentSize[:]), currentSize))

	cilinders := int64(binary.BigEndian.Uint16(header.DiskGeometry[:2]))
	heads := int64(header.DiskGeometry[2])
	sectors := int64(header.DiskGeometry[3])
	dsize := cilinders * heads * sectors * 512
	fmtField("Disk geometry",
		fmt.Sprintf("%s (c: %d, h: %d, s: %d) (%d bytes)",
			hexs(header.DiskGeometry[:]),
			cilinders,
			heads,
			sectors,
			dsize))

	fmtField("Disk type",
		fmt.Sprintf("%s (%s)", hexs(header.DiskType[:]), header.DiskTypeStr()))

	fmtField("Checksum", hexs(header.Checksum[:]))
	fmtField("UUID", uuid(header.UniqueId[:]))
	fmtField("Saved state", fmt.Sprintf("%d", header.SavedState[0]))

	return header
}

// Return the number of blocks in the disk, diskSize in bytes
func getMaxTableEntries(diskSize uint64) uint64 {
	return diskSize * (2 * 1024 * 1024) // block size is 2M
}

func hexToField(hexs string, field []byte) {
	h, err := hex.DecodeString(hexs)
	check(err)

	copy(field, h)
}

func CreateSparseVHD(size uint64, name string) {
	header := VHDHeader{}
	hexToField(VHD_COOKIE, header.Cookie[:])
	hexToField("00000002", header.Features[:])
	hexToField("00010000", header.FileFormatVersion[:])
	hexToField("0000000000000200", header.DataOffset[:])

	t := uint32(time.Now().Unix() - 946684800)
	binary.BigEndian.PutUint32(header.Timestamp[:], t)
	hexToField(VHD_CREATOR_APP, header.CreatorApplication[:])
	hexToField(VHD_CREATOR_HOST_OS, header.CreatorHostOS[:])
	binary.BigEndian.PutUint64(header.OriginalSize[:], size)
	binary.BigEndian.PutUint64(header.CurrentSize[:], size)

	// total sectors = disk size / 512b sector size
	totalSectors := math.Floor(float64(size / 512))
	// [C, H, S]
	geometry := calculateCHS(uint64(totalSectors))
	binary.BigEndian.PutUint16(header.DiskGeometry[:2], uint16(geometry[0]))
	header.DiskGeometry[2] = uint8(geometry[1])
	header.DiskGeometry[3] = uint8(geometry[2])

	hexToField("00000003", header.DiskType[:]) // Sparse 0x00000003
	hexToField("00000000", header.Checksum[:])
	copy(header.UniqueId[:], uuidgenBytes())

	header.addChecksum()

	// Fill the sparse header
	header2 := VHDExtraHeader{}
	hexToField(VHD_DYN_COOKIE, header2.Cookie[:])
	hexToField("ffffffffffffffff", header2.DataOffset[:])
	// header size + sparse header size
	binary.BigEndian.PutUint64(header2.TableOffset[:], uint64(VHD_EXTRA_HEADER_SIZE+VHD_HEADER_SIZE))
	hexToField("00010000", header2.HeaderVersion[:])

	maxTableSize := uint32(size / (VHD_BLOCK_SIZE))
	binary.BigEndian.PutUint32(header2.MaxTableEntries[:], maxTableSize)

	binary.BigEndian.PutUint32(header2.BlockSize[:], VHD_BLOCK_SIZE)
	binary.BigEndian.PutUint32(header2.ParentTimestamp[:], uint32(0))
	header2.addChecksum()

	f, err := os.Create(name)
	check(err)
	defer f.Close()

	binary.Write(f, binary.BigEndian, header)
	binary.Write(f, binary.BigEndian, header2)

	// Write BAT entries
	for count := uint32(0); count < maxTableSize; count += 1 {
		f.Write(bytes.Repeat([]byte{0xff}, 4))
	}

	// The BAT is always extended to a sector boundary
	// Windows creates 8K VHDs by default
	for count := uint32(0); count < (1536 - maxTableSize); count += 1 {
		f.Write(bytes.Repeat([]byte{0x0}, 4))
	}

	binary.Write(f, binary.BigEndian, header)
}

func PrintVHDHeaders(f *os.File) {
	vhdHeader := make([]byte, 512)
	_, err := f.Read(vhdHeader)
	check(err)
	header := readVHDHeader(vhdHeader)

	if header.DiskType[3] == 0x3 || header.DiskType[3] == 0x04 {
		fmt.Println("\nReading dynamic/differential VHD header...")
		readVHDExtraHeader(f)
	}
}
