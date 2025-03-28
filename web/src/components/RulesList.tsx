import React, { useEffect, useState } from 'react';
import { rulesApi } from '../services/api';
import { Button } from './ui';

interface Rule {
  id: string;
  name: string;
  description: string;
  sourceMetrics: string[];
  aggregationType: string;
  createdAt: string;
  updatedAt: string;
  active: boolean;
}

interface RulesListProps {
  onRuleSelect?: (rule: Rule) => void;
  onRefreshRequest?: () => void;
}

const RulesList: React.FC<RulesListProps> = ({ onRuleSelect, onRefreshRequest }) => {
  const [rules, setRules] = useState<Rule[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState<boolean>(false);
  const [deletingRuleIds, setDeletingRuleIds] = useState<string[]>([]);

  const fetchRules = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await rulesApi.getAllRules();
      
      if (response.error) {
        setError(response.error);
      } else if (response.data) {
        // Make sure each rule has a properly initialized sourceMetrics array
        const processedRules = response.data.map(rule => ({
          ...rule,
          // Convert either matcher.metric_names array or legacy sourceMetric to sourceMetrics array
          sourceMetrics: rule.matcher?.metric_names || 
                        (rule.sourceMetric ? [rule.sourceMetric] : [])
        }));
        setRules(processedRules);
      }
    } catch (err) {
      setError(`Failed to fetch rules: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRules();
  }, []);

  const handleRefresh = async () => {
    setRefreshing(true);
    await fetchRules();
    setRefreshing(false);
    if (onRefreshRequest) {
      onRefreshRequest();
    }
  };

  const handleDelete = async (ruleId: string) => {
    if (window.confirm('Are you sure you want to delete this rule?')) {
      setDeletingRuleIds(prev => [...prev, ruleId]);
      try {
        const response = await rulesApi.deleteRule(ruleId);
        
        if (response.error) {
          setError(response.error);
        } else {
          // Remove the deleted rule from the list
          setRules(rules.filter(rule => rule.id !== ruleId));
        }
      } catch (err) {
        setError(`Failed to delete rule: ${err instanceof Error ? err.message : String(err)}`);
      } finally {
        setDeletingRuleIds(prev => prev.filter(id => id !== ruleId));
      }
    }
  };

  return (
    <div className="bg-white shadow rounded-lg p-6">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-semibold text-gray-800">Aggregation Rules</h2>
        <Button
          onClick={handleRefresh}
          isLoading={refreshing}
          loadingText="Refreshing"
          size="small"
        >
          Refresh
        </Button>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}

      {loading ? (
        <div className="flex justify-center items-center h-20">
          <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500"></div>
        </div>
      ) : rules.length === 0 ? (
        <div className="text-center py-4 text-gray-600">
          No aggregation rules found. Create a rule to get started.
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Source Metrics</th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Aggregation</th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                <th scope="col" className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {rules.map((rule) => (
                <tr key={rule.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 whitespace-nowrap">
                    <div className="text-sm font-medium text-gray-900">{rule.name}</div>
                    <div className="text-xs text-gray-500">{rule.description}</div>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                    {rule.sourceMetrics.join(', ')}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                    {rule.aggregationType}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${rule.active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'}`}>
                      {rule.active ? 'Active' : 'Inactive'}
                    </span>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-right text-sm font-medium">
                    <Button
                      onClick={() => onRuleSelect && onRuleSelect(rule)}
                      variant="link"
                      size="small"
                      className="mr-3"
                    >
                      Edit
                    </Button>
                    <Button
                      onClick={() => handleDelete(rule.id)}
                      variant="text"
                      size="small"
                      isLoading={deletingRuleIds.includes(rule.id)}
                      loadingText="Deleting"
                      disabled={deletingRuleIds.includes(rule.id)}
                    >
                      Delete
                    </Button>
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

export default RulesList;