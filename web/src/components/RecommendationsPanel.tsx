import React, { useEffect, useState } from 'react';
import { recommendationsApi } from '../services/api';
import { Button } from './ui';

interface EstimatedImpact {
  cardinalityReduction: number;
  savingsPercentage: number;
  retentionPeriod?: string;
}

interface Rule {
  id: string;
  name: string;
  description: string;
  matcher: {
    metricNames: string[];
    labels?: Record<string, string>;
    labelRegex?: Record<string, string>;
  };
  aggregation: {
    type: string;
    intervalSeconds: number;
    segmentation: string[];
  };
  output: {
    metricName: string;
    additionalLabels?: Record<string, string>;
    dropOriginal: boolean;
  };
}

interface Recommendation {
  id: string;
  createdAt: string;
  metricName: string;
  cardinality: number;
  labelsToKeep: string[];
  confidence: number;
  status: 'pending' | 'applied' | 'rejected';
  rule: Rule;
  estimatedImpact: EstimatedImpact;
  source: string;
}

interface ResponseDataType {
  recommendations?: Recommendation[];
  total?: number;
}

const RecommendationsPanel: React.FC = () => {
  const [recommendations, setRecommendations] = useState<Recommendation[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [generating, setGenerating] = useState<boolean>(false);
  const [refreshing, setRefreshing] = useState<boolean>(false);
  const [processingRecommendationIds, setProcessingRecommendationIds] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const fetchRecommendations = async () => {
    setRefreshing(true);
    setError(null);
    setSuccessMessage(null);
    
    try {
      const response = await recommendationsApi.getAllRecommendations();
      
      if (response.error) {
        setError(response.error);
        setRecommendations([]);
      } else if (response.data) {
        // Check if data is an array directly or contains a recommendations property
        if (Array.isArray(response.data)) {
          setRecommendations(response.data);
        } else {
          // Cast response.data to ResponseDataType to access the recommendations property
          const responseData = response.data as ResponseDataType;
          if (responseData.recommendations && Array.isArray(responseData.recommendations)) {
            // Handle the backend API structure where recommendations are in a nested property
            setRecommendations(responseData.recommendations);
          } else {
            console.error("API returned invalid data format:", response.data);
            setRecommendations([]);
            setError("Received invalid data format from API");
          }
        }
      } else {
        // Handle case where neither error nor data is present
        setRecommendations([]);
        setError("No data received from API");
      }
    } catch (err) {
      setError(`Failed to fetch recommendations: ${err instanceof Error ? err.message : String(err)}`);
      setRecommendations([]);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    fetchRecommendations();
  }, []);

  const handleGenerateRecommendations = async () => {
    setGenerating(true);
    setError(null);
    setSuccessMessage(null);
    
    try {
      const response = await recommendationsApi.generateRecommendations();
      
      if (response.error) {
        setError(response.error);
      } else {
        setSuccessMessage('Successfully generated new recommendations');
        fetchRecommendations();
      }
    } catch (err) {
      setError(`Failed to generate recommendations: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      setGenerating(false);
    }
  };

  const handleApplyRecommendation = async (id: string) => {
    setProcessingRecommendationIds(prev => [...prev, id]);
    setError(null);
    setSuccessMessage(null);
    
    try {
      const response = await recommendationsApi.applyRecommendation(id);
      
      if (response.error) {
        setError(response.error);
      } else {
        setSuccessMessage('Successfully applied recommendation');
        // Update the status in the UI
        setRecommendations(prev => 
          prev.map(rec => rec.id === id ? { ...rec, status: 'applied' as const } : rec)
        );
      }
    } catch (err) {
      setError(`Failed to apply recommendation: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      setProcessingRecommendationIds(prev => prev.filter(recId => recId !== id));
    }
  };

  const handleRejectRecommendation = async (id: string) => {
    setProcessingRecommendationIds(prev => [...prev, id]);
    setError(null);
    setSuccessMessage(null);
    
    try {
      const response = await recommendationsApi.rejectRecommendation(id);
      
      if (response.error) {
        setError(response.error);
      } else {
        setSuccessMessage('Successfully rejected recommendation');
        // Update the status in the UI
        setRecommendations(prev => 
          prev.map(rec => rec.id === id ? { ...rec, status: 'rejected' as const } : rec)
        );
      }
    } catch (err) {
      setError(`Failed to reject recommendation: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      setProcessingRecommendationIds(prev => prev.filter(recId => recId !== id));
    }
  };

  const getStatusBadgeClass = (status: string) => {
    switch (status) {
      case 'applied':
        return 'bg-green-100 text-green-800';
      case 'rejected':
        return 'bg-red-100 text-red-800';
      default:
        return 'bg-yellow-100 text-yellow-800';
    }
  };

  return (
    <div className="bg-white shadow rounded-lg p-6">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-semibold text-gray-800">Metric Aggregation Recommendations</h2>
        <div className="flex gap-2">
          <Button
            onClick={fetchRecommendations}
            isLoading={refreshing}
            loadingText="Refreshing"
            variant="secondary"
            size="small"
          >
            Refresh
          </Button>
          <Button
            onClick={handleGenerateRecommendations}
            isLoading={generating}
            loadingText="Generating"
            variant="primary"
            size="small"
          >
            Generate Recommendations
          </Button>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}

      {successMessage && (
        <div className="bg-green-50 border border-green-200 text-green-800 px-4 py-3 rounded mb-4">
          {successMessage}
        </div>
      )}

      {loading ? (
        <div className="flex justify-center items-center h-32">
          <div className="animate-spin rounded-full h-10 w-10 border-t-2 border-b-2 border-indigo-500"></div>
        </div>
      ) : recommendations.length === 0 ? (
        <div className="text-center py-8 text-gray-600">
          <p>No recommendations available.</p>
          <p className="mt-2 text-sm">
            Generate recommendations based on your metric usage patterns to reduce cardinality.
          </p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Metric</th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Cardinality</th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Suggested Labels</th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Impact</th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Confidence</th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                <th scope="col" className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {recommendations.map((rec) => (
                <tr key={rec.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 whitespace-nowrap text-sm font-medium text-gray-900">
                    {rec.metricName}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                    {rec.cardinality.toLocaleString()}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500">
                    <div className="flex flex-wrap gap-1">
                      {rec.labelsToKeep.map(label => (
                        <span 
                          key={label}
                          className="inline-flex items-center bg-gray-100 text-gray-800 px-2 py-0.5 rounded-md text-xs"
                        >
                          {label}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                    {rec.estimatedImpact.cardinalityReduction.toLocaleString()} ({rec.estimatedImpact.savingsPercentage.toFixed(2)}%)
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                    {(rec.confidence * 100).toFixed(2)}%
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${getStatusBadgeClass(rec.status)}`}>
                      {rec.status.charAt(0).toUpperCase() + rec.status.slice(1)}
                    </span>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-right text-sm font-medium">
                    {rec.status === 'pending' && (
                      <>
                        <Button
                          onClick={() => handleApplyRecommendation(rec.id)}
                          variant="link"
                          size="small"
                          isLoading={processingRecommendationIds.includes(rec.id)}
                          loadingText="Applying"
                          disabled={processingRecommendationIds.includes(rec.id) || rec.status !== 'pending'}
                          className="mr-3"
                        >
                          Apply
                        </Button>
                        <Button
                          onClick={() => handleRejectRecommendation(rec.id)}
                          variant="text"
                          size="small"
                          isLoading={processingRecommendationIds.includes(rec.id)}
                          loadingText="Rejecting"
                          disabled={processingRecommendationIds.includes(rec.id) || rec.status !== 'pending'}
                        >
                          Reject
                        </Button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};

export default RecommendationsPanel;