package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/foxglove/mcap/go/mcap"
)

func tokenName(t mcap.TokenType) string {
	switch t {
	case mcap.TokenHeader:
		return "Header"
	case mcap.TokenFooter:
		return "Footer"
	case mcap.TokenSchema:
		return "Schema"
	case mcap.TokenChannel:
		return "Channel"
	case mcap.TokenMessage:
		return "Message"
	case mcap.TokenChunk:
		return "Chunk"
	case mcap.TokenMessageIndex:
		return "MessageIndex"
	case mcap.TokenChunkIndex:
		return "ChunkIndex"
	case mcap.TokenAttachmentIndex:
		return "AttachmentIndex"
	case mcap.TokenStatistics:
		return "Statistics"
	case mcap.TokenMetadata:
		return "Metadata"
	case mcap.TokenMetadataIndex:
		return "MetadataIndex"
	case mcap.TokenSummaryOffset:
		return "SummaryOffset"
	case mcap.TokenDataEnd:
		return "DataEnd"
	case mcap.TokenError:
		return "Error"
	case mcap.TokenInvalidChunk:
		return "InvalidChunk"
	default:
		return fmt.Sprintf("Unknown(%d)", int(t))
	}
}

func describe(t mcap.TokenType, payload []byte) string {
	switch t {
	case mcap.TokenHeader:
		h, err := mcap.ParseHeader(payload)
		if err != nil {
			return fmt.Sprintf("parse error: %v", err)
		}
		return fmt.Sprintf("profile=%q library=%q", h.Profile, h.Library)
	case mcap.TokenFooter:
		f, err := mcap.ParseFooter(payload)
		if err != nil {
			return fmt.Sprintf("parse error: %v", err)
		}
		return fmt.Sprintf("summaryStart=%d summaryOffsetStart=%d summaryCRC=0x%08x",
			f.SummaryStart, f.SummaryOffsetStart, f.SummaryCRC)
	case mcap.TokenSchema:
		s, err := mcap.ParseSchema(payload)
		if err != nil {
			return fmt.Sprintf("parse error: %v", err)
		}
		return fmt.Sprintf("id=%d name=%q encoding=%q dataLen=%d", s.ID, s.Name, s.Encoding, len(s.Data))
	case mcap.TokenChannel:
		c, err := mcap.ParseChannel(payload)
		if err != nil {
			return fmt.Sprintf("parse error: %v", err)
		}
		return fmt.Sprintf("id=%d schemaID=%d topic=%q encoding=%q", c.ID, c.SchemaID, c.Topic, c.MessageEncoding)
	case mcap.TokenMessage:
		m, err := mcap.ParseMessage(payload)
		if err != nil {
			return fmt.Sprintf("parse error: %v", err)
		}
		return fmt.Sprintf("channelID=%d seq=%d logTime=%d publishTime=%d dataLen=%d",
			m.ChannelID, m.Sequence, m.LogTime, m.PublishTime, len(m.Data))
	case mcap.TokenChunk:
		c, err := mcap.ParseChunk(payload)
		if err != nil {
			return fmt.Sprintf("parse error: %v", err)
		}
		return fmt.Sprintf("messageStart=%d messageEnd=%d compression=%q uncompressedSize=%d compressedRecordsLen=%d",
			c.MessageStartTime, c.MessageEndTime, c.Compression, c.UncompressedSize, len(c.Records))
	case mcap.TokenMessageIndex:
		mi, err := mcap.ParseMessageIndex(payload)
		if err != nil {
			return fmt.Sprintf("parse error: %v", err)
		}
		return fmt.Sprintf("channelID=%d entries=%d", mi.ChannelID, len(mi.Records))
	case mcap.TokenChunkIndex:
		ci, err := mcap.ParseChunkIndex(payload)
		if err != nil {
			return fmt.Sprintf("parse error: %v", err)
		}
		return fmt.Sprintf("chunkStart=%d chunkLength=%d messageStart=%d messageEnd=%d compression=%q",
			ci.ChunkStartOffset, ci.ChunkLength, ci.MessageStartTime, ci.MessageEndTime, ci.Compression)
	case mcap.TokenAttachmentIndex:
		ai, err := mcap.ParseAttachmentIndex(payload)
		if err != nil {
			return fmt.Sprintf("parse error: %v", err)
		}
		return fmt.Sprintf("offset=%d length=%d name=%q", ai.Offset, ai.Length, ai.Name)
	case mcap.TokenStatistics:
		s, err := mcap.ParseStatistics(payload)
		if err != nil {
			return fmt.Sprintf("parse error: %v", err)
		}
		return fmt.Sprintf("messages=%d schemas=%d channels=%d chunks=%d attachments=%d metadata=%d",
			s.MessageCount, s.SchemaCount, s.ChannelCount, s.ChunkCount, s.AttachmentCount, s.MetadataCount)
	case mcap.TokenMetadata:
		m, err := mcap.ParseMetadata(payload)
		if err != nil {
			return fmt.Sprintf("parse error: %v", err)
		}
		return fmt.Sprintf("name=%q entries=%d", m.Name, len(m.Metadata))
	case mcap.TokenMetadataIndex:
		mi, err := mcap.ParseMetadataIndex(payload)
		if err != nil {
			return fmt.Sprintf("parse error: %v", err)
		}
		return fmt.Sprintf("offset=%d length=%d name=%q", mi.Offset, mi.Length, mi.Name)
	case mcap.TokenSummaryOffset:
		so, err := mcap.ParseSummaryOffset(payload)
		if err != nil {
			return fmt.Sprintf("parse error: %v", err)
		}
		return fmt.Sprintf("groupOpcode=0x%02x groupStart=%d groupLength=%d",
			byte(so.GroupOpcode), so.GroupStart, so.GroupLength)
	case mcap.TokenDataEnd:
		de, err := mcap.ParseDataEnd(payload)
		if err != nil {
			return fmt.Sprintf("parse error: %v", err)
		}
		return fmt.Sprintf("dataSectionCRC=0x%08x", de.DataSectionCRC)
	default:
		return fmt.Sprintf("payloadLen=%d", len(payload))
	}
}

func run(path string, expandChunks bool) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	idx := 0
	printRow := func(name, detail string) {
		fmt.Printf("%-6d %-16s %s\n", idx, name, detail)
		idx++
	}

	lexer, err := mcap.NewLexer(f, &mcap.LexerOptions{
		EmitChunks:        !expandChunks,
		EmitInvalidChunks: true,
		AttachmentCallback: func(ar *mcap.AttachmentReader) error {
			printRow("Attachment", fmt.Sprintf("name=%q mediaType=%q logTime=%d createTime=%d dataSize=%d",
				ar.Name, ar.MediaType, ar.LogTime, ar.CreateTime, ar.DataSize))
			return nil
		},
	})
	if err != nil {
		return fmt.Errorf("new lexer: %w", err)
	}
	defer lexer.Close()

	fmt.Printf("# mcap record order: %s (expandChunks=%v)\n", path, expandChunks)
	fmt.Printf("%-6s %-16s %s\n", "INDEX", "RECORD", "DETAIL")

	var buf []byte
	for {
		tok, payload, err := lexer.Next(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("lexer.Next: %w", err)
		}
		buf = payload
		printRow(tokenName(tok), describe(tok, payload))
	}
	fmt.Printf("# total records: %d\n", idx)
	return nil
}

func main() {
	expandChunks := flag.Bool("expand-chunks", false, "decompress chunks and emit their inner records inline")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [--expand-chunks] <file.mcap>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(flag.Arg(0), *expandChunks); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
