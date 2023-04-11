package signing

import (
	"context"
	"fmt"

	txsigning "cosmossdk.io/x/tx/signing"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

// VerifySignature verifies a transaction signature contained in SignatureData abstracting over different signing modes
// and single vs multi-signatures.
func VerifySignature(ctx context.Context, pubKey cryptotypes.PubKey, signerData SignerData, sigData signing.SignatureData, handler SignModeHandler, tx sdk.Tx) error {
	switch data := sigData.(type) {
	case *signing.SingleSignatureData:
		signBytes, err := GetSignBytesWithContext(handler, ctx, data.SignMode, signerData, tx)
		if err != nil {
			return err
		}
		if !pubKey.VerifySignature(signBytes, data.Signature) {
			return fmt.Errorf("unable to verify single signer signature")
		}
		return nil

	case *signing.MultiSignatureData:
		multiPK, ok := pubKey.(multisig.PubKey)
		if !ok {
			return fmt.Errorf("expected %T, got %T", (multisig.PubKey)(nil), pubKey)
		}
		err := multiPK.VerifyMultisignature(func(mode signing.SignMode) ([]byte, error) {
			handlerWithContext, ok := handler.(SignModeHandlerWithContext)
			if ok {
				return handlerWithContext.GetSignBytesWithContext(ctx, mode, signerData, tx)
			}
			return handler.GetSignBytes(mode, signerData, tx)
		}, data)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unexpected SignatureData %T", sigData)
	}
}

// VerifySignatureV2 verifies a transaction signature contained in SignatureData abstracting over different signing
// modes. It differs from VerifySignature in that it uses the new txsigning.TxData interface in x/tx.
func VerifySignatureV2(
	ctx context.Context,
	pubKey cryptotypes.PubKey,
	signerData txsigning.SignerData,
	signatureData signing.SignatureData,
	handler txsigning.SignModeHandler,
	txData txsigning.TxData) error {

	switch data := signatureData.(type) {
	case *signing.SingleSignatureData:
		signBytes, err := handler.GetSignBytes(ctx, signerData, txData)
		if err != nil {
			return err
		}
		if !pubKey.VerifySignature(signBytes, data.Signature) {
			return fmt.Errorf("unable to verify single signer signature")
		}
		return nil

	case *signing.MultiSignatureData:
		multiPK, ok := pubKey.(multisig.PubKey)
		if !ok {
			return fmt.Errorf("expected %T, got %T", (multisig.PubKey)(nil), pubKey)
		}
		err := multiPK.VerifyMultisignature(func(mode signing.SignMode) ([]byte, error) {
			return handler.GetSignBytes(ctx, signerData, txData)
		}, data)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unexpected SignatureData %T", signatureData)
	}
}

// GetSignBytesWithContext gets the sign bytes from the sign mode handler. It
// checks if the sign mode handler supports SignModeHandlerWithContext, in
// which case it passes the context.Context argument. Otherwise, it fallbacks
// to GetSignBytes.
func GetSignBytesWithContext(h SignModeHandler, ctx context.Context, mode signing.SignMode, data SignerData, tx sdk.Tx) ([]byte, error) { //nolint:revive // refactor this in a future pr
	hWithCtx, ok := h.(SignModeHandlerWithContext)
	if ok {
		return hWithCtx.GetSignBytesWithContext(ctx, mode, data, tx)
	}
	return h.GetSignBytes(mode, data, tx)
}
