#!/bin/bash

# Configuration
FOLDER="/Users/jzucca/Desktop/QV"
API_URL="http://98.80.211.122:4000/upload"
OUTPUT_DIR="transcripts"
RESULTS_FILE="results.csv"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Initialize CSV with headers
echo "filename,id,status,timestamp" > "$RESULTS_FILE"

echo "Starting batch processing..."
echo "================================"

# Loop through all files in the folder
for file in "$FOLDER"/*; do
  # Skip if not a file (e.g., directories)
  if [ ! -f "$file" ]; then
    continue
  fi
  
  # Extract filename
  filename=$(basename "$file")
  
  echo "Processing: $filename"
  
  # Send request
  response=$(curl -s -X POST -F "audio=@$file" "$API_URL")
  
  # Check if curl succeeded
  if [ $? -ne 0 ]; then
    echo "❌ Error: curl failed for $filename"
    echo "$filename,ERROR,curl_failed,$(date)" >> "$RESULTS_FILE"
    continue
  fi
  
  # Parse response (extract id)
  call_id=$(echo "$response" | jq -r '.id')
  
  # Check if parsing succeeded
  if [ -z "$call_id" ] || [ "$call_id" == "null" ]; then
    echo "❌ Error: Invalid response for $filename"
    echo "Response: $response"
    echo "$filename,ERROR,invalid_response,$(date)" >> "$RESULTS_FILE"
    continue
  fi
  
  # Log success to CSV
  echo "$filename,$call_id,success,$(date)" >> "$RESULTS_FILE"
  
  echo "✅ Success: Saved as call ID $call_id"
  echo ""
done

echo "================================"
echo "Done! Results saved to $RESULTS_FILE"
echo "Total files processed: $(grep -c success $RESULTS_FILE)"
