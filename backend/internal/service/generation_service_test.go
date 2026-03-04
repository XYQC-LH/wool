package service

import "testing"

func TestResolveResourcePoolProviderKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "exact_banana", input: "banana", want: "banana"},
		{name: "case_insensitive", input: "Banana", want: "banana"},
		{name: "banana_dash_suffix", input: "banana-pro", want: "banana"},
		{name: "banana_underscore_suffix", input: "banana_pro", want: "banana"},
		{name: "banana_colon_suffix", input: "banana:pro-vt", want: "banana"},
		{name: "banana_slash_suffix", input: "banana/pro", want: "banana"},
		{name: "banana_at_suffix", input: "banana@vip", want: "banana"},
		{name: "nano_banana_dash", input: "nano-banana-pro", want: "banana"},
		{name: "exact_seedream", input: "seedream", want: "seedream"},
		{name: "seedream_dash_suffix", input: "seedream-4", want: "seedream"},
		{name: "exact_sora2", input: "sora2", want: "sora2"},
		{name: "sora2_dash_suffix", input: "sora2-prod", want: "sora2"},
		{name: "exact_jimeng", input: "jimeng", want: "jimeng"},
		{name: "jimeng_slash_suffix", input: "jimeng/pool-1", want: "jimeng"},
		{name: "empty", input: "", wantErr: true},
		{name: "spaces", input: "   ", wantErr: true},
		{name: "non_matching_prefix", input: "bananarama", wantErr: true},
		{name: "unknown", input: "openai", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveResourcePoolProviderKey(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (got=%q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected result: got=%q want=%q", got, tc.want)
			}
		})
	}
}
