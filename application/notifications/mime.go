package notifications

import (
	"bytes"
	"embed"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"strings"

	"jaytaylor.com/html2text"

	"github.com/silinternational/cover-api/log"
	"github.com/silinternational/cover-api/public"
)

var images = map[string]string{
	"beach_access":   "beach_access.png",
	"check":          "check_circle.png",
	"clipboard":      "clipboard.png",
	"clock":          "clock.png",
	"do_not_enter":   "do_not_enter.png",
	"dollar":         "dollar.png",
	"error":          "error.png",
	"logo":           "logo.png",
	"new":            "new.png",
	"signature_logo": "SIL_Glyph_Logo_Andika_RGB.png",
}

// rawEmail generates a multi-part MIME email message with a plain text, html text, and inline logo attachment as
// follows:
//
// * multipart/alternative
//   * text/plain
//   * multipart/related
//     * text/html
//     * image/png
//
// Abbreviated example of the generated email message:
//  From: from@example.com
//	To: to@example.com
//	Subject: subject text
//	Content-Type: multipart/alternative; boundary="boundary_alternative"
//
//	--boundary_alternative
//	Content-Type: text/plain; charset=utf-8
//
//	Plain text body
//	--boundary_alternative
//	Content-type: multipart/related; boundary="boundary_related"
//
//	--boundary_related
//	Content-Type: text/html; charset=utf-8
//
//	HTML body
//	--boundary_related
//	Content-Type: image/png
//	Content-Transfer-Encoding: base64
//	Content-ID: <logo>
//	--boundary_related--
//	--boundary_alternative--
func rawEmail(to, from, subject, body string) []byte {
	tbody, err := html2text.FromString(body)
	if err != nil {
		log.Warning("error converting html email to plain text,", err)
		tbody = body
	}

	b := &bytes.Buffer{}

	b.WriteString("From: " + from + "\n")
	b.WriteString("To: " + to + "\n")
	b.WriteString("Subject: " + subject + "\n")
	b.WriteString("MIME-Version: 1.0\n")

	alternativeWriter := multipart.NewWriter(b)
	b.WriteString(`Content-Type: multipart/alternative; type="text/plain"; boundary="` +
		alternativeWriter.Boundary() + `"` + "\n\n")

	w, err := alternativeWriter.CreatePart(textproto.MIMEHeader{
		"Content-Type":        {"text/plain; charset=utf-8"},
		"Content-Disposition": {"inline"},
	})
	if err != nil {
		log.Error("failed to create MIME text part,", err)
	} else {
		_, _ = fmt.Fprint(w, tbody)
	}

	relatedWriter := multipart.NewWriter(b)
	_, err = alternativeWriter.CreatePart(textproto.MIMEHeader{
		"Content-Type": {`multipart/related; type="text/html"; boundary="` + relatedWriter.Boundary() + `"`},
	})
	if err != nil {
		log.Error("failed to create MIME related part,", err)
	}

	w, err = relatedWriter.CreatePart(textproto.MIMEHeader{
		"Content-Type":        {"text/html; charset=utf-8"},
		"Content-Disposition": {"inline"},
	})
	if err != nil {
		log.Error("failed to create MIME html part,", err)
	} else {
		_, _ = fmt.Fprint(w, body)
	}

	cids := findImagesInBody(body)
	attachImages(relatedWriter, b, cids)

	if err = relatedWriter.Close(); err != nil {
		log.Error("failed to close MIME related part,", err)
	}

	if err = alternativeWriter.Close(); err != nil {
		log.Error("failed to close MIME alternative part,", err)
	}

	return b.Bytes()
}

func findImagesInBody(body string) map[string]string {
	imagesFound := map[string]string{}
	for cid, filename := range images {
		if strings.Contains(body, fmt.Sprintf(`src="cid:%s"`, cid)) {
			imagesFound[cid] = filename
		}
	}
	return imagesFound
}

func attachImages(relatedWriter *multipart.Writer, b *bytes.Buffer, images map[string]string) {
	efs := public.EFS()

	for cid, filename := range images {
		_, err := relatedWriter.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {"image/png"},
			"Content-Disposition":       {"inline"},
			"Content-ID":                {"<" + cid + ">"},
			"Content-Transfer-Encoding": {"base64"},
		})
		if err != nil {
			log.Errorf("failed to create MIME image part for %s, %s", cid, err)
			break
		}

		if err := encodeFile(&efs, filename, b); err != nil {
			log.Error(err)
		}
	}
}

// encodeFile reads a file from an embedded file system, base64 encodes it, and streams into a bytes.Buffer
func encodeFile(fs *embed.FS, filename string, buffer *bytes.Buffer) error {
	file, err := fs.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read %s file, %w", filename, err)
	}

	encoder := base64.NewEncoder(base64.StdEncoding, buffer)
	_, err = encoder.Write(file)
	if err != nil {
		return fmt.Errorf("failed to encode file %s, %w", filename, err)
	}

	err = encoder.Close()
	if err != nil {
		return fmt.Errorf("failed to close %s base64 encoder, %w", filename, err)
	}

	return nil
}
