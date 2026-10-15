# Historical audit evidence — v1 only

audit_20261015/repro_test.go and results.txt are preserved at baseline
35f6f75b05f37c4b3aadcd5e9f15cab89a53a719. Their PASS means a vulnerability was
reproduced, NOT that the implementation was safe. They reference the retired v1
API and are not intended to compile against v2. To repeat them, use an isolated
checkout of that baseline and an isolated MySQL database.

Go ignores testdata during normal ./... discovery. The production safety
assertions are dlock/*_test.go, including explicit integration-tagged tests;
CI runs those on MySQL 8.0 and 8.4. Do not add the historical probes to safety CI.
AUDIT.md is the historical report; REMEDIATION.md records v2 closure.
