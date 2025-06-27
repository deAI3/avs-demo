### 1. Aggregator

Feature:
- Store operators when register, delete when deregister. Stored info:
    - Operator pubkey
    - Stake amount
- Choose operator for generate and validate (demo: random generators, rest are validators)
- Receive operator applications for task then finalize application
- Receive result, store temp result then send back to query

Function:
- Listen for Register/Deregister
- Listen for Query
- Listen for Task Response
- Config for listening

Test:
- Fetch application for task
- Fetch 2 operators's return temp result 
- Fetch validate result


### 2. Operator

Function:
- Register/Deregister
- Config for listening
- Listen for Task Request, latest task result
- Call API for result
- Apply for task
- Submit result

Test:
- Fetch task
- Fetch latest task result
- Get result from model API 
- Validate success


### 3. Task

TaskRequest:
- Token Threshold
- Generation Operators num
- Expire time


State:
- WaitingForApply
- TempResult{current_step, tokens, generators}
- Finalize

TaskResponse:
- Result
- Generators
- Validators