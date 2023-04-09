package config

import (
	"fmt"
)

func addVars[T comparable](inVars map[string]T, outVars map[string]any) {
	for k, v := range inVars {
		//var zero T
		if fmt.Sprint(v) == "" { //v == zero { //Excluding all zero values might not be optimal for items like ints.
			continue
		}
		outVars[k] = v
	}
}

func arrayContains(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}
