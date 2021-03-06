package archive

import (
	"context"
	"io"

	ocilayout "github.com/containers/image/oci/layout"
	"github.com/containers/image/types"
	digest "github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

type ociArchiveImageSource struct {
	ref         ociArchiveReference
	unpackedSrc types.ImageSource
	tempDirRef  tempDirOCIRef
}

// newImageSource returns an ImageSource for reading from an existing directory.
// newImageSource untars the file and saves it in a temp directory
func newImageSource(ctx *types.SystemContext, ref ociArchiveReference, requestedManifestMIMETypes []string) (types.ImageSource, error) {
	tempDirRef, err := createUntarTempDir(ref)
	if err != nil {
		return nil, errors.Wrap(err, "error creating temp directory")
	}

	unpackedSrc, err := tempDirRef.ociRefExtracted.NewImageSource(ctx, requestedManifestMIMETypes)
	if err != nil {
		if err := tempDirRef.deleteTempDir(); err != nil {
			return nil, errors.Wrapf(err, "error deleting temp directory", tempDirRef.tempDirectory)
		}
		return nil, err
	}
	return &ociArchiveImageSource{ref: ref,
		unpackedSrc: unpackedSrc,
		tempDirRef:  tempDirRef}, nil
}

// LoadManifestDescriptor loads the manifest
func LoadManifestDescriptor(imgRef types.ImageReference) (imgspecv1.Descriptor, error) {
	ociArchRef, ok := imgRef.(ociArchiveReference)
	if !ok {
		return imgspecv1.Descriptor{}, errors.Errorf("error typecasting, need type ociArchiveReference")
	}
	tempDirRef, err := createUntarTempDir(ociArchRef)
	if err != nil {
		return imgspecv1.Descriptor{}, errors.Wrap(err, "error creating temp directory")
	}
	defer tempDirRef.deleteTempDir()

	descriptor, err := ocilayout.LoadManifestDescriptor(tempDirRef.ociRefExtracted)
	if err != nil {
		return imgspecv1.Descriptor{}, errors.Wrap(err, "error loading index")
	}
	return descriptor, nil
}

// Reference returns the reference used to set up this source.
func (s *ociArchiveImageSource) Reference() types.ImageReference {
	return s.ref
}

// Close removes resources associated with an initialized ImageSource, if any.
// Close deletes the temporary directory at dst
func (s *ociArchiveImageSource) Close() error {
	defer s.tempDirRef.deleteTempDir()
	return s.unpackedSrc.Close()
}

// GetManifest returns the image's manifest along with its MIME type
// (which may be empty when it can't be determined but the manifest is available).
func (s *ociArchiveImageSource) GetManifest() ([]byte, string, error) {
	return s.unpackedSrc.GetManifest()
}

func (s *ociArchiveImageSource) GetTargetManifest(digest digest.Digest) ([]byte, string, error) {
	return s.unpackedSrc.GetTargetManifest(digest)
}

// GetBlob returns a stream for the specified blob, and the blob's size.
func (s *ociArchiveImageSource) GetBlob(info types.BlobInfo) (io.ReadCloser, int64, error) {
	return s.unpackedSrc.GetBlob(info)
}

func (s *ociArchiveImageSource) GetSignatures(c context.Context) ([][]byte, error) {
	return s.unpackedSrc.GetSignatures(c)
}
