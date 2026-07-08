// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"fmt"

	"oblikovati.org/api/wire"
)

// cylinderFaces returns every cylindrical face of the active part, each carrying its reference
// key and representative point (the face's range-box centre). A headed fastener's shank is a
// cylinder; a plain-cylindrical head (socket cap screw) is another, so callers disambiguate on
// the point rather than assuming a single cylinder.
func (b *PartBuilder) cylinderFaces() ([]wire.TopologyRef, error) {
	keys, err := b.api.Model().ReferenceKeys()
	if err != nil {
		return nil, fmt.Errorf("read reference keys: %w", err)
	}
	var cyl []wire.TopologyRef
	for _, body := range keys.Bodies {
		for _, f := range body.Faces {
			if f.Kind == "cylinder" {
				cyl = append(cyl, f)
			}
		}
	}
	return cyl, nil
}

// CylinderFaceKey returns the reference key of the part's single cylindrical face — the shank
// side of a headed fastener whose head is *not* a cylinder (a hex bolt), which the thread
// feature targets. It errors unless exactly one cylinder exists, so an ambiguous (two
// cylinders) or missing shank is caught rather than silently threading the wrong surface.
func (b *PartBuilder) CylinderFaceKey() (string, error) {
	cyl, err := b.cylinderFaces()
	if err != nil {
		return "", err
	}
	if len(cyl) != 1 {
		return "", fmt.Errorf("want exactly 1 cylindrical face for the shank, found %d", len(cyl))
	}
	return cyl[0].Key, nil
}

// ShankCylinderFaceKey returns the reference key of the shank of a headed screw whose head may
// *also* be a cylinder (a socket-head cap screw has a cylindrical head, a countersunk one a
// conical head). The shank always extends deepest along the build axis (the head grows from the
// top plane at z=0 and the shank continues below it), so the shank is the cylindrical face whose
// representative point sits lowest in z. It errors when no cylinder exists.
func (b *PartBuilder) ShankCylinderFaceKey() (string, error) {
	cyl, err := b.cylinderFaces()
	if err != nil {
		return "", err
	}
	if len(cyl) == 0 {
		return "", fmt.Errorf("no cylindrical face for the shank; a headed screw needs one")
	}
	return lowestFace(cyl).Key, nil
}

// lowestFace returns the face whose representative point has the smallest z (deepest along the
// build axis). faces must be non-empty.
func lowestFace(faces []wire.TopologyRef) wire.TopologyRef {
	lowest := faces[0]
	for _, f := range faces[1:] {
		if faceZ(f) < faceZ(lowest) {
			lowest = f
		}
	}
	return lowest
}

// faceZ is the z of a face's representative point (0 when the point is absent/degenerate).
func faceZ(f wire.TopologyRef) float64 {
	if len(f.Point) < 3 {
		return 0
	}
	return f.Point[2]
}
