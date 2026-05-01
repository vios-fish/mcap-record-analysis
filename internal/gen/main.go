package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/foxglove/mcap/go/mcap"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <out.mcap>\n", os.Args[0])
		os.Exit(2)
	}
	f, err := os.Create(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w, err := mcap.NewWriter(f, &mcap.WriterOptions{
		Chunked:     true,
		ChunkSize:   1024,
		Compression: mcap.CompressionZSTD,
	})
	if err != nil {
		panic(err)
	}

	if err := w.WriteHeader(&mcap.Header{Profile: "demo", Library: "mcap-record-order/gen"}); err != nil {
		panic(err)
	}
	if err := w.WriteSchema(&mcap.Schema{ID: 1, Name: "Foo", Encoding: "jsonschema", Data: []byte(`{}`)}); err != nil {
		panic(err)
	}
	if err := w.WriteChannel(&mcap.Channel{ID: 1, SchemaID: 1, Topic: "/foo", MessageEncoding: "json"}); err != nil {
		panic(err)
	}
	if err := w.WriteChannel(&mcap.Channel{ID: 2, SchemaID: 1, Topic: "/bar", MessageEncoding: "json"}); err != nil {
		panic(err)
	}
	for i := 0; i < 5; i++ {
		ch := uint16(1 + i%2)
		if err := w.WriteMessage(&mcap.Message{
			ChannelID:   ch,
			Sequence:    uint32(i),
			LogTime:     uint64(i * 1_000_000),
			PublishTime: uint64(i * 1_000_000),
			Data:        []byte(fmt.Sprintf(`{"i":%d}`, i)),
		}); err != nil {
			panic(err)
		}
	}
	attachData := []byte("hello")
	if err := w.WriteAttachment(&mcap.Attachment{
		Name:      "note.txt",
		MediaType: "text/plain",
		LogTime:   1,
		DataSize:  uint64(len(attachData)),
		Data:      bytes.NewReader(attachData),
	}); err != nil {
		panic(err)
	}
	if err := w.WriteMetadata(&mcap.Metadata{Name: "info", Metadata: map[string]string{"k": "v"}}); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
}
