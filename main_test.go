package main

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)


type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

func captureStdout(t *testing.T, run func()) string {
	t.Helper()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}

	os.Stdout = writer
	defer func() {
		os.Stdout = oldStdout
	}()

	run()

	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout pipe writer: %v", err)
	}
	os.Stdout = oldStdout

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close stdout pipe reader: %v", err)
	}
	return string(output)
}

func Test_IsAnswerCorrect(t *testing.T) {
	question := Question{CorrectAnswer: "red"}

	tests := []struct {
		name     string
		answer   string
		expected bool
	}{
		{name: "exact answer", answer: "red", expected: true},
		{name: "different case", answer: "RED", expected: true},
		{name: "incorrect answer", answer: "blue", expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isAnswerCorrect(question, test.answer); got != test.expected {
				t.Errorf("isAnswerCorrect(%q) = %t, want %t", test.answer, got, test.expected)
			}
		})
	}
}

func Test_GetAnswer(t *testing.T) {
	scannerErr := errors.New("input failed")
	tests := []struct {
		name       string
		question   Question
		input      io.Reader
		wantAnswer string
		wantErr    error
		wantOutput string
	}{
		{
			name:       "trims input and formats prompt",
			question:   Question{Prompt: "Pick a color", Options: []string{"red", "blue"}},
			input:      strings.NewReader("  red  \n"),
			wantAnswer: "red",
			wantOutput: "Pick a color\nOptions:\n  - red\n  - blue\n> ",
		},
		{
			name:       "returns EOF when input is empty",
			question:   Question{},
			input:      strings.NewReader(""),
			wantErr:    io.EOF,
			wantOutput: "\nOptions:\n> ",
		},
		{
			name:       "returns scanner error",
			question:   Question{},
			input:      errorReader{err: scannerErr},
			wantErr:    scannerErr,
			wantOutput: "\nOptions:\n> ",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var answer string
			var answerErr error
			output := captureStdout(t, func() {
				input := bufio.NewScanner(test.input)
				answer, answerErr = getAnswer(test.question, input)
			})

			if answer != test.wantAnswer {
				t.Errorf("getAnswer returned %q, want %q", answer, test.wantAnswer)
			}
			if test.wantErr != nil {
				if !errors.Is(answerErr, test.wantErr) {
					t.Errorf("getAnswer error = %v, want %v", answerErr, test.wantErr)
				}
			} else if answerErr != nil {
				t.Errorf("getAnswer returned an unexpected error: %v", answerErr)
			}
			if output != test.wantOutput {
				t.Errorf("getAnswer output = %q, want %q", output, test.wantOutput)
			}
		})
	}
}

func runMainWithInput(t *testing.T, input string) string {
	t.Helper()

	oldStdin := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}

	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = reader.Close()
		_ = writer.Close()
	})

	if _, err := io.WriteString(writer, input); err != nil {
		t.Fatalf("write quiz input: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close stdin pipe writer: %v", err)
	}

	return captureStdout(t, main)
}


func Test_QuizIntegration(t *testing.T) {
	output := runMainWithInput(t, "red\nnot-blue\n")

	for _, want := range []string{
		"first color?",
		"second color?",
		"Phase: Correct\tMessage: well done",
		"Phase: Incorrect\tMessage: Best luck on next",
		"Phase: Done",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("quiz output does not contain %q:\n%s", want, output)
		}
	}

	if strings.Index(output, "second color?") < strings.Index(output, "first color?") {
		t.Errorf("second question appeared before first:\n%s", output)
	}
}
