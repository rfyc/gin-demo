package utils

import "runtime"

func GetFunctionName() string {
	pc, _, _, _ := runtime.Caller(1)
	funcObj := runtime.FuncForPC(pc)
	return funcObj.Name()
}
