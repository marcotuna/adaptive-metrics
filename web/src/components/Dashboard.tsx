import React, { useState } from 'react';
import RulesList from './RulesList';
import RuleForm from './RuleForm';
import RecommendationsPanel from './RecommendationsPanel';
import MetricsUsagePanel from './MetricsUsagePanel';
import { Button } from './ui';

type TabType = 'rules' | 'recommendations' | 'metrics-usage';

const Dashboard: React.FC = () => {
  const [activeTab, setActiveTab] = useState<TabType>('rules');
  const [showRuleForm, setShowRuleForm] = useState(false);

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center mb-6">
        <h1 className="text-3xl font-bold text-gray-800 mb-4 md:mb-0">Adaptive Metrics Dashboard</h1>
        {activeTab === 'rules' && (
          <Button 
            onClick={() => setShowRuleForm(true)} 
            variant="primary"
          >
            Create New Rule
          </Button>
        )}
      </div>
      
      <div className="mb-6 border-b border-gray-200">
        <nav className="-mb-px flex space-x-4">
          <button
            onClick={() => setActiveTab('rules')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'rules'
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Rules
          </button>
          <button
            onClick={() => setActiveTab('recommendations')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'recommendations'
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Recommendations
          </button>
          <button
            onClick={() => setActiveTab('metrics-usage')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'metrics-usage'
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Metrics Usage
          </button>
        </nav>
      </div>
      
      {/* Active tab content */}
      {activeTab === 'rules' && <RulesList />}
      
      {activeTab === 'recommendations' && <RecommendationsPanel />}
      
      {activeTab === 'metrics-usage' && <MetricsUsagePanel />}
      
      {/* Rule form modal */}
      {showRuleForm && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg shadow-lg max-w-2xl w-full">
            <RuleForm onClose={() => setShowRuleForm(false)} />
          </div>
        </div>
      )}
    </div>
  );
};

export default Dashboard;