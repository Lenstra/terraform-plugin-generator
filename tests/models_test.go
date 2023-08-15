package tests

import (
	"testing"

	structs "github.com/Lenstra/terraform-plugin-generator/tests/structs"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/stretchr/testify/require"
)

func TestEncoding(t *testing.T) {
	config := &structs.Config{}
	data, diags := EncodeConfig(config)
	require.False(t, diags.HasError())

	var roundTrip *structs.Config
	diags = decodeConfig(path.Empty(), data, &roundTrip)
	require.False(t, diags.HasError())

	require.Equal(t, config, roundTrip)
}
