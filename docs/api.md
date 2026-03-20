# API

## POST

### /reportCorpus

Request Body:
```json
{
    "fuzzer": "AFL",
    "identity": "master_1",
    "corpus": [
        "/path/to/corpus1",
        "/path/to/corpus2"
    ],
    "period": "begin"
}
```

**Fields:**
- `fuzzer` (string, required): The name of the fuzzer reporting the corpus.
- `identity` (string, required): Unique identifier for the fuzzer instance.
- `corpus` (array of strings, required): List of paths to corpus files or directories.
- `period` (string, required): Indicates the reporting period. Use `"begin"` to mark the start of a fuzzing cycle and save the initial coverage baseline for the fuzzer.

Response Body:
```json
{
  "success": true,
  "message": "Corpus report received successfully. Processing in background.",
  "data": {
    "task_id": "554f3a04-01d8-4292-b55f-4b8fd06d6bed"
  }
}
```

### /log

Request Body:
```json
{
    "log": "log content"
}
```


## GET
### /peekResult

Response Body:
```json
{
  "success": true,
  "message": "Constraint groups retrieved",
  "data": {
    "constraint_groups": [
      {
        "group_id": "1ca34b37-d225-4944-b981-ab74157e8842",
        "function": "png_handle_sCAL",
        "importance": 0.483546069753371,
        "paths":[ ["LLVMFuzzerTestOneInput", "png_read_info"] ],
        "constraint_score": {
          "comp_opr": 0.483546069753371,
          "arith_opr": 0.0,
          "bitwise_opr": 0.0,
          "rel_opr": 0.0
        }
      }
    ],
    "fuzzer_scores": {
      "AFL": {
        "comp_opr": 40.0,
        "arith_opr": 30.0,
        "bitwise_opr": 20.0,
        "rel_opr": 10.0
      }
    },
    "fuzzer_cov_inc": {
      "AFL": 100
    }
  }
}
```
