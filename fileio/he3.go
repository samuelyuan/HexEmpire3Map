package fileio

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"io"
	"io/ioutil"
	"log"
)

type MapStyle struct {
	Grass     byte
	Mountains byte
	Desert    byte
	Sea       byte
	Light     byte
}

type FieldType byte

const (
	Grass FieldType = iota
	Sand
	Farmland
	Forest
	Snow
	Airport
	Factory
	Town
	City
	Capital
)

const (
	ELEVATION_MOUNTAIN = 0.6
)

var (
	SERIALIZATION_TYPE_CONV = [10]int{0, 1, 2, 3, 4, 9, 5, 6, 7, 8}
)

type MapTile struct {
	Height       float32
	IsSea        bool
	IsMountain   bool
	HasRoad      bool
	HasFlag      bool
	TileType     FieldType
	CityName     string
	Party        int
	HasInfantry  bool
	HasArtillery bool
	Infantry     *Army
	Artillery    *Army
}

type Army struct {
	X             int32
	Y             int32
	UnitInfantry  int32
	UnitArtillery int32
	Morale        float32
}

type HE3Map struct {
	MapTiles  [][]*MapTile
	MapTitle  string
	MapAuthor string
	MapStyle  MapStyle
	Width     int32
	Depth     int32
}

func readString(streamReader *io.SectionReader) (string, error) {
	stringLength := byte(0)
	if err := binary.Read(streamReader, binary.LittleEndian, &stringLength); err != nil {
		return "", err
	}
	byteData := make([]byte, stringLength)
	if err := binary.Read(streamReader, binary.LittleEndian, &byteData); err != nil {
		return "", err
	}
	return string(byteData), nil
}

func writeString(buffer *bytes.Buffer, str string) {
	// Need to append string length to the file
	// Strings can't be over 255 bytes
	buffer.WriteByte(byte(len(str)))
	buffer.WriteString(str)
}

func writeFloat32(buffer *bytes.Buffer, f float32) {
	err := binary.Write(buffer, binary.LittleEndian, f)
	if err != nil {
		log.Fatal("Failed to convert float32 to bytes", err)
	}
}

func writeInteger(buffer *bytes.Buffer, num int32) {
	err := binary.Write(buffer, binary.LittleEndian, num)
	if err != nil {
		log.Fatal("Failed to convert int32 to bytes", err)
	}
}

func serializeArmy(buffer *bytes.Buffer, army *Army) {
	writeInteger(buffer, army.X)
	writeInteger(buffer, army.Y)
	writeInteger(buffer, army.UnitInfantry)
	writeInteger(buffer, army.UnitArtillery)
	writeFloat32(buffer, army.Morale)
}

func Serialize(mapData *HE3Map) string {
	buffer := new(bytes.Buffer)
	writeString(buffer, "hexmap")
	writeInteger(buffer, int32(7))
	writeString(buffer, mapData.MapTitle)
	writeString(buffer, mapData.MapAuthor)
	writeInteger(buffer, mapData.Width)
	writeInteger(buffer, mapData.Depth)
	buffer.WriteByte(mapData.MapStyle.Grass)
	buffer.WriteByte(mapData.MapStyle.Mountains)
	buffer.WriteByte(mapData.MapStyle.Desert)
	buffer.WriteByte(mapData.MapStyle.Sea)
	buffer.WriteByte(mapData.MapStyle.Light)
	for x := 0; x < int(mapData.Width); x++ {
		for y := 0; y < int(mapData.Depth); y++ {
			field := mapData.MapTiles[x][y]
			writeFloat32(buffer, field.Height)
			flags := byte(SERIALIZATION_TYPE_CONV[int(field.TileType)])
			if field.HasRoad {
				flags += 64
			}
			if field.HasFlag {
				flags += 128
			}
			buffer.WriteByte(flags)
			if field.TileType >= Airport {
				writeString(buffer, field.CityName)
			}
			writeInteger(buffer, int32(field.Party))
			if field.Infantry != nil {
				buffer.WriteByte(1) // true
				serializeArmy(buffer, field.Infantry)
			} else {
				buffer.WriteByte(0) // false
			}
			if field.Artillery != nil {
				buffer.WriteByte(1) // true
				serializeArmy(buffer, field.Artillery)
			} else {
				buffer.WriteByte(0) // false
			}
		}
	}

	// Game state
	buffer.WriteByte(0) // false

	compressed := Compress(buffer.Bytes())
	return base64.StdEncoding.EncodeToString(compressed)
}

func DeserializeArmy(
	streamReader *io.SectionReader,
	party int,
	gameState bool,
	version int,
	thumb bool,
) *Army {
	x := int32(0)
	if err := binary.Read(streamReader, binary.LittleEndian, &x); err != nil {
		log.Fatal("Error reading x: ", err)
	}

	y := int32(0)
	if err := binary.Read(streamReader, binary.LittleEndian, &y); err != nil {
		log.Fatal("Error reading y: ", err)
	}

	unitInfantry := int32(0)
	if err := binary.Read(streamReader, binary.LittleEndian, &unitInfantry); err != nil {
		log.Fatal("Error reading unitInfantry: ", err)
	}

	unitArtillery := int32(0)
	if version > 1 {
		if err := binary.Read(streamReader, binary.LittleEndian, &unitArtillery); err != nil {
			log.Fatal("Error reading unitArtillery: ", err)
		}
	}

	morale := float32(0)
	if err := binary.Read(streamReader, binary.LittleEndian, &morale); err != nil {
		log.Fatal("Error reading morale: ", err)
	}

	return &Army{
		X:             x,
		Y:             y,
		UnitInfantry:  unitInfantry,
		UnitArtillery: unitArtillery,
		Morale:        morale,
	}
}

func Deserialize(content []byte) *HE3Map {
	rawDecodedText, err := base64.StdEncoding.DecodeString(string(content))
	if err != nil {
		log.Fatal("Failed to decode string: ", err)
	}
	inputData := Decompress(rawDecodedText)

	streamReader := io.NewSectionReader(bytes.NewReader(inputData), int64(0), int64(len(inputData)))

	version1, err := readString(streamReader)
	if err != nil || version1 != "hexmap" {
		log.Fatal("The header string "+version1+" is the wrong string", err)
	}

	version2 := int32(0)
	if err := binary.Read(streamReader, binary.LittleEndian, &version2); err != nil {
		log.Fatal("Error reading map version2: ", err)
	}

	mapTitle, err := readString(streamReader)
	if err != nil {
		log.Fatal("Error reading map title: ", err)
	}

	mapAuthor, err := readString(streamReader)
	if err != nil {
		log.Fatal("Error reading map author: ", err)
	}

	width := int32(0)
	if err := binary.Read(streamReader, binary.LittleEndian, &width); err != nil {
		log.Fatal("Error reading width: ", err)
	}

	depth := int32(0)
	if err := binary.Read(streamReader, binary.LittleEndian, &depth); err != nil {
		log.Fatal("Error reading depth: ", err)
	}

	style := MapStyle{}
	if version2 >= 5 {
		if err := binary.Read(streamReader, binary.LittleEndian, &style); err != nil {
			log.Fatal("Error reading style: ", err)
		}
	}

	// TODO: Figure out correct value
	thumb := false

	tileMap := make([][]*MapTile, int(width))
	for x := 0; x < int(width); x++ {
		tileMap[x] = make([]*MapTile, int(depth))
		for z := 0; z < int(depth); z++ {
			tile := MapTile{}

			height := float32(0)
			if err := binary.Read(streamReader, binary.LittleEndian, &height); err != nil {
				log.Fatal("Error reading height: ", err)
			}
			tile.Height = height
			if tile.Height <= 0.0 {
				tile.IsSea = true
			} else {
				tile.IsSea = false
			}
			if tile.Height >= ELEVATION_MOUNTAIN {
				tile.IsMountain = true
			} else {
				tile.IsMountain = false
			}

			num := byte(0)
			if err := binary.Read(streamReader, binary.LittleEndian, &num); err != nil {
				log.Fatal("Error reading num: ", err)
			}
			tile.HasRoad = false
			if (int(num) & 64) == 64 {
				tile.HasRoad = true
				num -= byte(64)
			}
			tile.HasFlag = false
			if (int(num) & 128) == 128 {
				tile.HasFlag = true
				num -= byte(128)
			}
			tile.TileType = Grass
			for i := 0; i < len(SERIALIZATION_TYPE_CONV); i++ {
				if SERIALIZATION_TYPE_CONV[i] == int(num) {
					tile.TileType = FieldType(i)
				}
			}
			if tile.TileType >= Airport {
				cityName, err := readString(streamReader)
				tile.CityName = cityName
				if err != nil {
					log.Fatal("Error reading city name: ", err)
				}
			}
			party := int32(0)
			if err := binary.Read(streamReader, binary.LittleEndian, &party); err != nil {
				log.Fatal("Error reading party: ", err)
			}
			tile.Party = int(party)

			if tile.HasFlag {
				// TODO: set party flag
			}
			boolArmy := byte(0)
			if err := binary.Read(streamReader, binary.LittleEndian, &boolArmy); err != nil {
				log.Fatal("Error reading boolArmy: ", err)
			}
			if boolArmy == 1 {
				army := DeserializeArmy(streamReader, int(party), false, int(version2), thumb)
				tile.HasInfantry = true
				tile.Infantry = army
			} else {
				tile.HasInfantry = false
				tile.Infantry = nil
			}

			boolArtillery := byte(0)
			if err := binary.Read(streamReader, binary.LittleEndian, &boolArtillery); err != nil {
				log.Fatal("Error reading boolArtillery: ", err)
			}
			if boolArtillery == 1 {
				tile.HasArtillery = true
			} else {
				tile.HasArtillery = false
			}
			if version2 >= 3 && boolArtillery == 1 {
				artillery := DeserializeArmy(streamReader, int(party), false, int(version2), thumb)
				tile.Artillery = artillery
			} else {
				tile.Artillery = nil
			}

			tileMap[x][z] = &tile
		}
	}

	boolGameState := byte(0)
	if err := binary.Read(streamReader, binary.LittleEndian, &boolGameState); err != nil {
		log.Fatal("Error reading boolGameState: ", err)
	}

	return &HE3Map{
		MapTiles:  tileMap,
		MapTitle:  mapTitle,
		MapAuthor: mapAuthor,
		MapStyle:  style,
		Width:     width,
		Depth:     depth,
	}
}

func ReadHE3File(filename string) [][]*MapTile {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal("Failed to load map: ", err)
	}

	mapData := Deserialize(content)

	return mapData.MapTiles
}

func DecompressHE3File(filename string) []byte {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal("Failed to load map: ", err)
	}
	rawDecodedText, err := base64.StdEncoding.DecodeString(string(content))
	if err != nil {
		log.Fatal("Failed to decode string: ", err)
	}
	return Decompress(rawDecodedText)
}
