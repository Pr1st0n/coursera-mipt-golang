package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// сюда писать код
func ExecutePipeline(jobs ...job) {
	in := make(chan interface{}, 0) // Only required as a placeholder for the first job.
	out := make(chan interface{})
	_runJob := func(job0 job, in0, out0 chan interface{}) {
		job0(in0, out0)
		close(out0) // Break channel range loop when job is finished.
	}

	// Initialize all the jobs.
	for _, job0 := range jobs {
		go _runJob(job0, in, out)
		in = out // Bypass data.
		out = make(chan interface{})
	}

	for range in { // Wait until last job is finished.
		<-in
	}
}

// crc32 считается через функцию DataSignerCrc32
// md5 считается через DataSignerMd5
func SingleHash(in, out chan interface{}) {
	// SingleHash считает значение crc32(data)+"~"+crc32(md5(data))
	// ( конкатенация двух строк через ~), где data - то что пришло на вход
	// (по сути - числа из первой функции)
	for raw := range in {
		num, ok := raw.(int)

		if !ok {
			fmt.Println("SingleHash: Error: can't convert input to integer")
		}

		str := strconv.Itoa(num)
		result := DataSignerCrc32(str) + "~" + DataSignerCrc32(DataSignerMd5(str))

		out <- result
	}
}

func MultiHash(in, out chan interface{}) {
	// MultiHash считает значение crc32(th+data))
	// (конкатенация цифры, приведённой к строке и строки),
	// где th=0..5 ( т.е. 6 хешей на каждое входящее значение ),
	// потом берёт конкатенацию результатов в порядке расчета (0..5),
	// где data - то что пришло на вход (и ушло на выход из SingleHash)
	for raw := range in {
		str, ok := raw.(string)

		if !ok {
			fmt.Println("MultiHash: Error: can't convert input to string")
		}

		result := ""

		for th := 0; th < 6; th++ {
			result += DataSignerCrc32(strconv.Itoa(th) + str)
		}

		out <- result
	}
}

func CombineResults(in, out chan interface{}) {
	// CombineResults получает все результаты, сортирует (https://golang.org/pkg/sort/),
	// объединяет отсортированный результат через _ (символ подчеркивания) в одну строку
	var values []string

	for raw := range in {
		str, ok := raw.(string)

		if !ok {
			fmt.Println("CombineResults: Error: can't convert input to integer cmb")
		}

		values = append(values, str)
	}

	sort.Strings(values)

	out <- strings.Join(values, "_")
}
