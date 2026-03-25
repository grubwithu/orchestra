# API

## POST

### /reportCorpus

Request Body:
```json
{
    "fuzzer": "AFL",
    "job_id": 1,
    "job_budget": 100,
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
- `job_id` (integer, required): Unique identifier for the fuzzing job.
- `job_budget` (integer, required): Budget for the fuzzing job.
- `identity` (string, required): Unique identifier for the fuzzer instance.
- `corpus` (array of strings, required): List of paths to corpus files or directories.
- `period` (string, required): Indicates the reporting period. Use `"begin"` to mark the start of a fuzzing cycle and save the initial coverage baseline for the fuzzer.

Response Body:
```json
{
  "success": true,
  "message": "Corpus report received successfully. Processing in background."
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
  "message": "Results retrieved",
  "data": {
    "plugin_results": {
      "fuzzer": {
        "fuzzer_scores": {
          "AFL": {
            "val_cmp": 0.5,
            "bit_opr": 0.3,
            "str_mat": 0.8,
            "art_opr": 0.2,
            "comp_opr": 0.6
          }
        }
      },
      "seed": {
        "constraint_group": {
          "group_id": "1ca34b37-d225-4944-b981-ab74157e8842",
          "path": ["LLVMFuzzerTestOneInput", "png_read_info", "png_handle_sCAL"],
          "leaf_function": "png_handle_sCAL",
          "file_name": "png.c",
          "importance": 0.483546069753371,
          "constraint_score": {
            "val_cmp": 0.0,
            "bit_opr": 0.0,
            "str_mat": 0.0,
            "art_opr": 0.0,
            "comp_opr": 1.0
          }
        }
      }
    }
  }
}
```
