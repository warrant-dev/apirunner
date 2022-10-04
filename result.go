package apirunner

import "fmt"

type TestResult struct {
	Passed bool
	Name   string
	Output []string
}

func Failed(name string, output []string) TestResult {
	return TestResult{
		Passed: false,
		Name:   name,
		Output: output,
	}
}

func Passed(name string) TestResult {
	return TestResult{
		Passed: true,
		Name:   name,
		Output: nil,
	}
}

func (result TestResult) PrintResult() {
	if result.Passed {
		fmt.Printf("%-40s%30s\n", result.Name, "PASSED")
	} else {
		fmt.Printf("%-40s%30s\n", result.Name, "FAILED")
		for _, output := range result.Output {
			fmt.Printf("\t%s\n", output)
		}
	}
}

func (result TestResult) TabbedResult() []string {
	resultString := make([]string, 0)
	if result.Passed {
		resultString = append(resultString, fmt.Sprintf("%s:\tPASSED\n", result.Name))
	} else {
		resultString = append(resultString, fmt.Sprintf("%s:\tFAILED\n", result.Name))
		for _, output := range result.Output {
			resultString = append(resultString, fmt.Sprintf("\t%s\n", output))
		}
	}
	return resultString
}
