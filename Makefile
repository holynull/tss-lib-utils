test_dkg_eddsa:
	GOLOG_LOG_LEVEL="debug" go test -v -count=1 -timeout 120s -run TestDKG github.com/holynull/tss-lib-utils/eddsa/utils
test_sign_eddsa:
	GOLOG_LOG_LEVEL="debug" go test -v -count=1 -timeout 120s -run TestSign github.com/holynull/tss-lib-utils/eddsa/utils
test_resharing_eddsa:
	GOLOG_LOG_LEVEL="debug" go test -v -count=1 -timeout 120s -run TestResharing github.com/holynull/tss-lib-utils/eddsa/utils
test_dkg_ecdsa:
	GOLOG_LOG_LEVEL="debug" go test -v -count=1 -timeout 120s -run TestDKG github.com/holynull/tss-lib-utils/ecdsa/utils
test_sign_ecdsa:
	GOLOG_LOG_LEVEL="debug" go test -v -count=1 -timeout 120s -run TestSign github.com/holynull/tss-lib-utils/ecdsa/utils
test_resharing_ecdsa:
	GOLOG_LOG_LEVEL="debug" go test -v -count=1 -timeout 120s -run TestResharing github.com/holynull/tss-lib-utils/ecdsa/utils