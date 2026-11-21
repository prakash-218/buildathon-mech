package auto

// Primitive tuple types (Rust tuples). Fields x0, x1, ... in element order.

type tupIF struct {
	x0 int64
	x1 float64
}
type tupFV struct {
	x0 float64
	x1 V3
}
type tupVF struct {
	x0 V3
	x1 float64
}
type tupFF struct {
	x0 float64
	x1 float64
}
type tupIV struct {
	x0 int64
	x1 V3
}
type tupFIV struct {
	x0 float64
	x1 int64
	x2 V3
}
type tupIFF struct {
	x0 int64
	x1 float64
	x2 float64
}
type tupFN struct {
	x0 float64
	x1 int
}
type tupFU struct {
	x0 float64
	x1 uint32
}
type tupFI struct {
	x0 float64
	x1 int64
}
type tupVFF struct {
	x0 V3
	x1 float64
	x2 float64
}
type tupWF struct {
	x0 V2
	x1 float64
}
type tupNN struct {
	x0 int
	x1 int
}
type tupNF struct {
	x0 int
	x1 float64
}
type tupNWB struct {
	x0 int
	x1 V2
	x2 bool
}
type tupNW struct {
	x0 int
	x1 V2
}
type tupUU struct {
	x0 uint32
	x1 uint32
}
type tupIU struct {
	x0 int64
	x1 uint32
}
type tupIFFU struct {
	x0 int64
	x1 float64
	x2 float64
	x3 uint32
}
type tupJJG struct {
	x0 int32
	x1 int32
	x2 float32
}
type tupFNN struct {
	x0 float64
	x1 int
	x2 int
}
type tupFIF struct {
	x0 float64
	x1 int64
	x2 float64
}
type tupFIVB struct {
	x0 float64
	x1 int64
	x2 V3
	x3 bool
}
type tupFFNN struct {
	x0 float64
	x1 float64
	x2 int
	x3 int
}
type tupFFU struct {
	x0 float64
	x1 float64
	x2 uint32
}
type tupFFF struct {
	x0 float64
	x1 float64
	x2 float64
}
type tupFVF struct {
	x0 float64
	x1 V3
	x2 float64
}
type tupFVBF struct {
	x0 float64
	x1 V3
	x2 bool
	x3 float64
}
type tupFW struct {
	x0 float64
	x1 V2
}
type tupGU struct {
	x0 float32
	x1 uint32
}
type tupBF struct {
	x0 bool
	x1 float64
}
type tupBB struct {
	x0 bool
	x1 bool
}
type tupVFB struct {
	x0 V3
	x1 float64
	x2 bool
}
type tupVV struct {
	x0 V3
	x1 V3
}
