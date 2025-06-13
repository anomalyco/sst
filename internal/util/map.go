package util

import "iter"

func Map[T, U any](seq iter.Seq[T], f func(T) U) iter.Seq[U] {
	return func(yield func(U) bool) {
		seq(func(a T) bool {
			return yield(f(a))
		})
	}
}
