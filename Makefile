test_dkg_eddsa:
	GOLOG_LOG_LEVEL="debug" go test -v -count=1 -timeout 300s -run TestDKG github.com/holynull/tss-lib-utils/eddsa/utils