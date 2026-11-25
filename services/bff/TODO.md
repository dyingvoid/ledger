# TODO

- Add rollback account creation endpoint. It must delete `account_info` if the ledger account does not exist.
- Add fetch account endpoint. It must fetch account info and the ledger account, and log an error if either is missing.
- Separate temporal worker into another application