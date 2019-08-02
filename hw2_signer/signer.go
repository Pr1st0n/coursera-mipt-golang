package main

import (
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

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

// SingleHash считает значение crc32(data)+"~"+crc32(md5(data))
// ( конкатенация двух строк через ~), где data - то что пришло на вход
// (по сути - числа из первой функции)
// crc32 считается через функцию DataSignerCrc32
// md5 считается через DataSignerMd5
func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	lock := make(chan int, 1)

	for raw := range in {
		wg.Add(1)
		num, ok := raw.(int)

		if !ok {
			fmt.Println("SingleHash: Error: can't convert input to integer")
		}

		str := strconv.Itoa(num)
		crc32 := make(chan string, 1)
		md5 := make(chan string, 1)

		// Do the parallel calculation of crc32(data).
		go func(ch chan string, str0 string) {
			ch <- DataSignerCrc32(str)
		}(crc32, str)

		// Do the parallel calculation of crc32(md5(data)).
		go func(ch chan string, str0 string) {
			// Limit parallel execution.
			lock <- 1
			md5Str := DataSignerMd5(str)
			<-lock

			md5 <- DataSignerCrc32(md5Str)
		}(md5, str)

		// Concat the result and pass further.
		go func(ch1 chan string, ch2 chan string, out chan interface{}, wg *sync.WaitGroup) {
			defer wg.Done()
			result := <-ch1 + "~" + <-ch2
			out <- result
		}(crc32, md5, out, wg)
	}

	wg.Wait()
	close(lock)
}

// MultiHash считает значение crc32(th+data))
// (конкатенация цифры, приведённой к строке и строки),
// где th=0..5 ( т.е. 6 хешей на каждое входящее значение ),
// потом берёт конкатенацию результатов в порядке расчета (0..5),
// где data - то что пришло на вход (и ушло на выход из SingleHash)
func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}

	for raw := range in {
		wg.Add(1)
		str, ok := raw.(string)

		if !ok {
			fmt.Println("MultiHash: Error: can't convert input to string")
		}

		ch := make(chan string, 6)

		runtime.GOMAXPROCS(6)

		for th := 0; th < 6; th++ {
			runtime.Gosched()
			// Do the parallel calculation.
			go func(th0 int, str0 string, ch0 chan string) {
				res := DataSignerCrc32(strconv.Itoa(th0) + str0)
				time.Sleep(time.Duration(th0) * 5 * time.Millisecond) // Preserve loop order.
				ch0 <- res
			}(th, str, ch)
		}

		// Collect result of calculations, concat it and pass further.
		go func(ch0 chan string, out0 chan interface{}) {
			defer wg.Done()
			defer close(ch)

			result := ""

			for th0 := 0; th0 < 6; th0++ {
				result += <-ch
			}

			out0 <- result
		}(ch, out)
	}

	wg.Wait()
}

// CombineResults получает все результаты, сортирует (https://golang.org/pkg/sort/),
// объединяет отсортированный результат через _ (символ подчеркивания) в одну строку
func CombineResults(in, out chan interface{}) {
	var values []string

	// Combine all of the results.
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
