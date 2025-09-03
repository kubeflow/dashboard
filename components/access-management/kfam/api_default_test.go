package kfam

import (
	"reflect"
	"testing"
)

func TestSanitizeClusterAdmins(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		out  []string
	}{
		{
			name: "empty slice",
			in:   []string{},
			out:  nil,
		},
		{
			name: "nil slice",
			in:   nil,
			out:  nil,
		},
		{
			name: "empty strings",
			in:   []string{"", ""},
			out:  nil,
		},
		{
			name: "whitespace only strings",
			in:   []string{"  ", " "},
			out:  nil,
		},
		{
			name: "valid admins",
			in:   []string{"user1@example.com", "user2@example.com"},
			out:  []string{"user1@example.com", "user2@example.com"},
		},
		{
			name: "admins with leading spaces",
			in:   []string{"  user1@example.com", " user2@example.com"},
			out:  []string{"user1@example.com", "user2@example.com"},
		},
		{
			name: "admins with trailing spaces",
			in:   []string{"user1@example.com  ", "user2@example.com "},
			out:  []string{"user1@example.com", "user2@example.com"},
		},
		{
			name: "admins with leading and trailing spaces",
			in:   []string{"  user1@example.com  ", " user2@example.com "},
			out:  []string{"user1@example.com", "user2@example.com"},
		},
		{
			name: "mixed empty, whitespace, and admins",
			in:   []string{"", "user1@example.com", "   ", "user2@example.com  "},
			out:  []string{"user1@example.com", "user2@example.com"},
		},
		{
			name: "duplicate admins",
			in:   []string{"user1@example.com", "user1@example.com"},
			out:  []string{"user1@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized := sanitizeClusterAdmins(tt.in)
			if !reflect.DeepEqual(sanitized, tt.out) {
				t.Errorf("output: %v, expected: %v", sanitized, tt.out)
			}
		})
	}
}
