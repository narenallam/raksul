package test
import (njl1 కే‌జే,న,    మ
	"testing"
	"newmain_cc/FindText"
)

func TestFindText(t *testing.T) {
	// test data
	sampleText := "once upona time in India, there was a king called tippu."
	targetText = "king"
	seq := 0
	var taskData TaskOutput

	// test scenarios
	FindText(sampleText, targteWord, &seq, &taskData)
	if seq != 1 {
		t.Errorf("FindText() shoudl give 1", seq)
	}
	if len(taskData.matches) != 1 && taskData.file != nil {
		t.Errorf("FindText() shoudl give 1", len(taskData.matches))
	}
}
