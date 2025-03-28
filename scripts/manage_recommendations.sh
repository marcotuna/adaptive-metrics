#!/bin/bash

# Script to manage Adaptive Metrics recommendations
# Usage: ./manage_recommendations.sh [command] [args]

BASE_URL="http://localhost:8080"
RECOMMENDATIONS_API="${BASE_URL}/api/v1/recommendations"

# Command: list - List all recommendations
list_recommendations() {
    echo "Listing all recommendations..."
    curl -s ${RECOMMENDATIONS_API} | jq '.'
}

# Command: generate - Generate new recommendations
generate_recommendations() {
    echo "Generating new recommendations based on current usage data..."
    curl -s -X POST ${RECOMMENDATIONS_API}/generate | jq '.'
}

# Command: get - Get a specific recommendation
get_recommendation() {
    if [ -z "$1" ]; then
        echo "Error: Recommendation ID is required"
        echo "Usage: $0 get [id]"
        exit 1
    fi
    
    echo "Getting recommendation with ID: $1"
    curl -s ${RECOMMENDATIONS_API}/$1 | jq '.'
}

# Command: apply - Apply a specific recommendation
apply_recommendation() {
    if [ -z "$1" ]; then
        echo "Error: Recommendation ID is required"
        echo "Usage: $0 apply [id]"
        exit 1
    fi
    
    echo "Applying recommendation with ID: $1"
    curl -s -X POST ${RECOMMENDATIONS_API}/$1/apply | jq '.'
}

# Command: reject - Reject a specific recommendation
reject_recommendation() {
    if [ -z "$1" ]; then
        echo "Error: Recommendation ID is required"
        echo "Usage: $0 reject [id]"
        exit 1
    fi
    
    echo "Rejecting recommendation with ID: $1"
    curl -s -X POST ${RECOMMENDATIONS_API}/$1/reject | jq '.'
}

# Command: apply-all - Apply all pending recommendations
apply_all_recommendations() {
    echo "Applying all pending recommendations..."
    
    # Get all recommendations
    recommendations=$(curl -s ${RECOMMENDATIONS_API} | jq -r '.recommendations[] | select(.status == "pending") | .id')
    
    if [ -z "$recommendations" ]; then
        echo "No pending recommendations found."
        return
    fi
    
    # Apply each recommendation
    for id in $recommendations; do
        echo "Applying recommendation: $id"
        curl -s -X POST ${RECOMMENDATIONS_API}/$id/apply > /dev/null
        echo "Done."
    done
    
    echo "All pending recommendations have been applied."
}

# Command: top - Show top recommendations by impact
show_top_recommendations() {
    limit=${1:-5}  # Default to showing top 5 if not specified
    
    echo "Showing top $limit recommendations by cardinality reduction impact..."
    
    # Get all recommendations and sort by impact
    curl -s ${RECOMMENDATIONS_API} | jq -r "
        .recommendations
        | sort_by(.estimated_impact.cardinality_reduction)
        | reverse
        | .[0:$limit]
        | .[]
        | \"ID: \\(.id) | Name: \\(.rule.name) | Cardinality Reduction: \\(.estimated_impact.cardinality_reduction) | Savings: \\(.estimated_impact.savings_percentage)%\"
    "
}

# Command: summary - Show a summary of recommendations
show_recommendations_summary() {
    echo "Recommendation Summary:"
    
    # Get summary counts
    summary=$(curl -s ${RECOMMENDATIONS_API} | jq -r '
        {
            total: .recommendations | length,
            pending: [.recommendations[] | select(.status == "pending")] | length,
            applied: [.recommendations[] | select(.status == "applied")] | length,
            rejected: [.recommendations[] | select(.status == "rejected")] | length,
            avg_impact: [.recommendations[].estimated_impact.savings_percentage] | add / length
        }
    ')
    
    echo "$summary" | jq '.'
}

# Main command router
case "$1" in
    list)
        list_recommendations
        ;;
    generate)
        generate_recommendations
        ;;
    get)
        get_recommendation "$2"
        ;;
    apply)
        apply_recommendation "$2"
        ;;
    reject)
        reject_recommendation "$2"
        ;;
    apply-all)
        apply_all_recommendations
        ;;
    top)
        show_top_recommendations "$2"
        ;;
    summary)
        show_recommendations_summary
        ;;
    *)
        echo "Adaptive Metrics Recommendation Manager"
        echo ""
        echo "Usage: $0 [command] [args]"
        echo ""
        echo "Available commands:"
        echo "  list                    - List all recommendations"
        echo "  generate                - Generate new recommendations based on current usage data"
        echo "  get [id]                - Get details for a specific recommendation"
        echo "  apply [id]              - Apply a specific recommendation"
        echo "  reject [id]             - Reject a specific recommendation"
        echo "  apply-all               - Apply all pending recommendations"
        echo "  top [limit]             - Show top recommendations by impact (default: 5)"
        echo "  summary                 - Show a summary of recommendations"
        echo ""
        echo "Example:"
        echo "  $0 apply rec-a1b2c3d4"
        ;;
esac