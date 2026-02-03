package operator

import "context"

type TxTest struct{}

func (t *TxTest) Commit(ctx context.Context) error   { return nil }
func (t *TxTest) Rollback(ctx context.Context) error { return nil }
