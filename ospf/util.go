package ospf

import "golang.org/x/exp/constraints"

func abs[T constraints.Signed](a T) T {
	if a < 0 {
		return -a
	} else {
		return a
	}
}
