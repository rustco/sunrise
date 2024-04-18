package malicious

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	tmrand "github.com/cometbft/cometbft/libs/rand"
	"github.com/stretchr/testify/require"
	"github.com/sunrise-zone/sunrise-app/pkg/appconsts"
	"github.com/sunrise-zone/sunrise-app/pkg/wrapper"
	"github.com/sunrise-zone/sunrise-app/test/util/blobfactory"
	"github.com/sunrise-zone/sunrise-app/test/util/testfactory"
	"github.com/sunrise-zone/sunrise-app/test/util/testnode"
)

// TestOutOfOrderNMT tests that the malicious NMT implementation is able to
// generate the same root as the ordered NMT implementation when the leaves are
// added in the same order and can generate roots when leaves are out of
// order.
func TestOutOfOrderNMT(t *testing.T) {
	squareSize := uint64(64)
	c := NewConstructor(squareSize)
	goodConstructor := wrapper.NewConstructor(squareSize)

	orderedTree := goodConstructor(0, 0)
	maliciousOrderedTree := c(0, 0)
	maliciousUnorderedTree := c(0, 0)
	data := testfactory.GenerateRandNamespacedRawData(64)

	// compare the roots generated by pushing ordered data
	for _, d := range data {
		err := orderedTree.Push(d)
		require.NoError(t, err)
		err = maliciousOrderedTree.Push(d)
		require.NoError(t, err)
	}

	goodOrderedRoot, err := orderedTree.Root()
	require.NoError(t, err)
	malOrderedRoot, err := maliciousOrderedTree.Root()
	require.NoError(t, err)
	require.Equal(t, goodOrderedRoot, malOrderedRoot)

	// test the new tree with unordered data
	for i := range data {
		j := tmrand.Intn(len(data))
		data[i], data[j] = data[j], data[i]
	}

	for _, d := range data {
		err := maliciousUnorderedTree.Push(d)
		require.NoError(t, err)
	}

	root, err := maliciousUnorderedTree.Root()
	require.NoError(t, err)
	require.Len(t, root, 90)                   // two namespaces + 32 bytes of hash = 90 bytes
	require.NotEqual(t, goodOrderedRoot, root) // quick sanity check to ensure the roots are different
}

// TestMaliciousTestNode runs a single validator network using the malicious
// node. This will begin to produce out of order blocks after block height of 5.
func TestMaliciousTestNode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping MaliciousTestNode in short mode.")
	}
	accounts := testfactory.RandomAccountNames(5)
	cfg := OutOfOrderNamespaceConfig(5).
		WithFundedAccounts(accounts...)

	cctx, _, _ := testnode.NewNetwork(t, cfg)
	_, err := cctx.WaitForHeight(6)
	require.NoError(t, err)

	// submit a multiblob tx where each blob is using a random namespace. This
	// will result in the first two blobs being swapped in the square as per the
	// malicious square builder.
	signer, err := testnode.NewSignerFromContext(cctx, accounts[0])
	require.NoError(t, err)
	blobs := blobfactory.ManyRandBlobs(tmrand.NewRand(), 10_000, 10_000, 10_000, 10_000, 10_000, 10_000, 10_000)
	txres, err := signer.SubmitPayForBlob(cctx.GoContext(), blobs, blobfactory.DefaultTxOpts()...)
	require.NoError(t, err)
	require.Equal(t, abci.CodeTypeOK, txres.Code)

	// fetch the block that included in the tx
	inclusionHeight := txres.Height
	block, err := cctx.Client.Block(cctx.GoContext(), &inclusionHeight)
	require.NoError(t, err)

	// check that we can recalculate the data root using the malicious code but
	// not the correct code
	_, err = Construct(block.Block.Txs.ToSliceOfBytes(), appconsts.LatestVersion, appconsts.DefaultSquareSizeUpperBound, OutOfOrderExport)
	require.Error(t, err)
}
