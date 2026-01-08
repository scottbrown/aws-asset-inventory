package awsassetinventory

import (
	"errors"
	"testing"
)

func TestRegionError_Error(t *testing.T) {
	re := RegionError{
		Region: Region("us-east-1"),
		Err:    errors.New("API error"),
	}

	got := re.Error()
	want := "[us-east-1] API error"
	if got != want {
		t.Errorf("RegionError.Error() = %v, want %v", got, want)
	}
}

func TestRegionError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	re := RegionError{
		Region: Region("us-west-2"),
		Err:    underlying,
	}

	if re.Unwrap() != underlying {
		t.Error("RegionError.Unwrap() should return underlying error")
	}
}

func TestCollectErrors_Error_Single(t *testing.T) {
	ce := CollectErrors{
		Errors: []RegionError{
			{Region: Region("us-east-1"), Err: errors.New("access denied")},
		},
	}

	got := ce.Error()
	want := "[us-east-1] access denied"
	if got != want {
		t.Errorf("CollectErrors.Error() single = %v, want %v", got, want)
	}
}

func TestCollectErrors_Error_Multiple(t *testing.T) {
	ce := CollectErrors{
		Errors: []RegionError{
			{Region: Region("us-east-1"), Err: errors.New("access denied")},
			{Region: Region("us-west-2"), Err: errors.New("timeout")},
		},
	}

	got := ce.Error()
	want := "2 regions failed: [us-east-1] access denied; [us-west-2] timeout"
	if got != want {
		t.Errorf("CollectErrors.Error() multiple = %v, want %v", got, want)
	}
}

func TestCollectErrors_Regions(t *testing.T) {
	ce := CollectErrors{
		Errors: []RegionError{
			{Region: Region("us-east-1"), Err: errors.New("error1")},
			{Region: Region("eu-west-1"), Err: errors.New("error2")},
			{Region: Region("ap-southeast-2"), Err: errors.New("error3")},
		},
	}

	regions := ce.Regions()

	if len(regions) != 3 {
		t.Fatalf("CollectErrors.Regions() length = %v, want 3", len(regions))
	}
	if regions[0] != "us-east-1" {
		t.Errorf("CollectErrors.Regions()[0] = %v, want us-east-1", regions[0])
	}
	if regions[1] != "eu-west-1" {
		t.Errorf("CollectErrors.Regions()[1] = %v, want eu-west-1", regions[1])
	}
	if regions[2] != "ap-southeast-2" {
		t.Errorf("CollectErrors.Regions()[2] = %v, want ap-southeast-2", regions[2])
	}
}

func TestRegionError_ErrorsAs(t *testing.T) {
	underlying := errors.New("underlying")
	re := RegionError{
		Region: Region("us-east-1"),
		Err:    underlying,
	}

	if !errors.Is(re, underlying) {
		t.Error("errors.Is should match underlying error through Unwrap")
	}
}
