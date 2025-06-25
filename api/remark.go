package zbz

import "zbz/flux"

// Flux is a global singleton flux instance (renamed from Remark)
var Flux = flux.Flux

// Deprecated: Use Flux instead
var Remark = flux.Flux
