package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"github.com/RobertMe/cert-watcher/pkg/subscriber"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
	"html/template"
	"strings"
)

type actionCopy struct {
	OnError          onErrorHandling
	Destination      string
	FileNameTemplate *template.Template
	Format           string
}

func newCopyAction(data map[string]string) *actionCopy {
	a := actionCopy{
		OnError: parseActionOnError(data),
		Format:  "PEM",
	}

	var ok bool
	if a.Destination, ok = data["destination"]; !ok {
		return nil
	}

	filename, ok := data["filename"]
	if !ok {
		filename = "{{.Domain}}.{{.Extension}}"
	}

	var err error
	a.FileNameTemplate, err = template.New("").Parse(strings.TrimSpace(filename))
	if err != nil {
		return nil
	}

	if format, ok := data["format"]; ok && format == "PEM" {
		a.Format = format
	}

	return &a
}

func (a *actionCopy) onError() onErrorHandling {
	return a.OnError
}

func (a *actionCopy) execute(invocation subscriber.Invocation, containerId string, client client.APIClient, ctx context.Context) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	logger := log.Ctx(ctx)

	defer func() {
		if err := tw.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close temporary tar file")
		}
	}()

	switch a.Format {
	case "PEM":
		err := a.createPemArchive(tw, invocation)
		if err != nil {
			return err
		}
	}

	err := client.CopyToContainer(ctx, containerId, a.Destination, &buf, dockertypes.CopyToContainerOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (a *actionCopy) createPemArchive(tw *tar.Writer, invocation subscriber.Invocation) error {
	certificate := invocation.Certificate
	if err := writeBytesToTar(tw, a.buildFileName(invocation, "crt"), certificate.Cert); err != nil {
		return err
	}

	if err := writeBytesToTar(tw, a.buildFileName(invocation, "key"), certificate.Key); err != nil {
		return err
	}

	return nil
}

func writeBytesToTar(tw *tar.Writer, fileName string, contents []byte) error {
	if fileName == "" {
		return errors.New("no file name provided")
	}

	hdr := &tar.Header{
		Name: fileName,
		Mode: 0600,
		Size: int64(len(contents)),
	}

	err := tw.WriteHeader(hdr)
	if err != nil {
		return err
	}

	_, err = tw.Write(contents)
	if err != nil {
		return err
	}

	return nil
}

func (a *actionCopy) buildFileName(invocation subscriber.Invocation, extension string) string {
	buf := bytes.NewBuffer([]byte{})
	err := a.FileNameTemplate.Execute(buf, map[string]string{
		"Domain":    invocation.Domain,
		"Extension": extension,
	})

	if err != nil {
		return ""
	}

	return buf.String()
}
