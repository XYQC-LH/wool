'use client';

import { useState, useEffect, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Modal } from '@/components/ui/modal';
import { Pagination } from '@/components/common/pagination';
import { Loading } from '@/components/common/loading';
import { EmptyState } from '@/components/common/empty-state';
import { ConfirmDialog } from '@/components/common/confirm-dialog';
import { useToast } from '@/components/ui/use-toast';
import api from '@/lib/api';

// 类型定义
interface ModelProvider {
  id: number;
  operation?: string;
  model_id: string;
  model_name: string;
  channel_id: number;
  channel_name: string;
  upstream_model_name?: string;
  actual_cost_per_1k_input: number;
  actual_cost_per_1k_output: number;
  priority: number;
  weight: number;
  is_cost_priority?: boolean;
  status: 'active' | 'disabled' | 'cooling' | 'circuit_open';
  circuit_state: 'closed' | 'open' | 'half_open';
  failure_count: number;
  failure_threshold: number;
  recovery_timeout_seconds?: number;
  connect_timeout_ms?: number;
  attempt_timeout_ms?: number;
  stream_first_chunk_timeout_ms?: number;
  health_score: number;
  total_requests: number;
  success_requests: number;
  failed_requests: number;
  created_at: string;
  updated_at: string;
}

interface Channel {
  id: number;
  name: string;
}

interface Model {
  id: string;
  name: string;
}

interface ProviderInstance {
  id: number;
  provider_id: number;
  provider_name?: string;
  name: string;
  instance_type: 'api_key' | 'resource_account' | 'session';
  resource_account_id?: number;
  account_name?: string;
  weight: number;
  status: 'active' | 'disabled' | 'cooling';
  max_concurrency: number;
  rpm_limit: number;
  tpm_limit: number;
  total_requests: number;
  success_requests: number;
  failed_requests: number;
  success_rate: number;
  avg_latency_ms: number;
  created_at: string;
  updated_at: string;
}

interface ResourceAccount {
  id: number;
  account_name: string;
  channel_id: number;
  status: string;
}

// 状态标签组件
function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    active: 'bg-green-100 text-green-800',
    disabled: 'bg-gray-100 text-gray-800',
    cooling: 'bg-yellow-100 text-yellow-800',
    circuit_open: 'bg-red-100 text-red-800',
  };
  const labels: Record<string, string> = {
    active: '启用',
    disabled: '禁用',
    cooling: '冷却',
    circuit_open: '熔断',
  };
  return (
    <span className={`px-2 py-1 rounded-full text-xs font-medium ${colors[status] || 'bg-gray-100'}`}>
      {labels[status] || status}
    </span>
  );
}

// 熔断状态标签组件
function CircuitBadge({ state }: { state: string }) {
  const colors: Record<string, string> = {
    closed: 'bg-green-100 text-green-800',
    open: 'bg-red-100 text-red-800',
    half_open: 'bg-yellow-100 text-yellow-800',
  };
  const labels: Record<string, string> = {
    closed: '正常',
    open: '熔断',
    half_open: '半开',
  };
  return (
    <span className={`px-2 py-1 rounded-full text-xs font-medium ${colors[state] || 'bg-gray-100'}`}>
      {labels[state] || state}
    </span>
  );
}

// 健康分数组件
function HealthScore({ score }: { score: number }) {
  let color = 'text-green-600';
  if (score < 50) {
    color = 'text-red-600';
  } else if (score < 80) {
    color = 'text-yellow-600';
  }
  return (
    <span className={`font-medium ${color}`}>
      {Number(score).toFixed(1)}
    </span>
  );
}

function getErrorMessage(error: unknown): string {
  if (typeof error === 'string' && error.trim()) return error
  if (error instanceof Error && error.message.trim()) return error.message

  if (error && typeof error === 'object') {
    const record = error as Record<string, unknown>
    const response = record.response
    if (response && typeof response === 'object') {
      const responseRecord = response as Record<string, unknown>
      const data = responseRecord.data
      if (data && typeof data === 'object') {
        const dataRecord = data as Record<string, unknown>
        const message = dataRecord.message
        if (typeof message === 'string' && message.trim()) return message

        const nestedError = dataRecord.error
        if (nestedError && typeof nestedError === 'object') {
          const nestedMessage = (nestedError as Record<string, unknown>).message
          if (typeof nestedMessage === 'string' && nestedMessage.trim()) return nestedMessage
        }
      }
    }
  }

  return '请求失败'
}

export default function ProvidersPage() {
  const { toast } = useToast();
  const [providers, setProviders] = useState<ModelProvider[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [models, setModels] = useState<Model[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);

  // 筛选条件
  const [filterModelId, setFilterModelId] = useState('all');
  const [filterChannelId, setFilterChannelId] = useState('all');
  const [filterStatus, setFilterStatus] = useState('all');
  const [filterCircuitState, setFilterCircuitState] = useState('all');

  // 模态框状态
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [showCircuitModal, setShowCircuitModal] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [selectedProvider, setSelectedProvider] = useState<ModelProvider | null>(null);

  // 实例管理状态
  const [showInstancesModal, setShowInstancesModal] = useState(false);
  const [showCreateInstanceModal, setShowCreateInstanceModal] = useState(false);
  const [showEditInstanceModal, setShowEditInstanceModal] = useState(false);
  const [showInstanceDeleteConfirm, setShowInstanceDeleteConfirm] = useState(false);
  const [instances, setInstances] = useState<ProviderInstance[]>([]);
  const [instancesLoading, setInstancesLoading] = useState(false);
  const [instancesTotal, setInstancesTotal] = useState(0);
  const [instancesPage, setInstancesPage] = useState(1);
  const [selectedInstance, setSelectedInstance] = useState<ProviderInstance | null>(null);
  const [resourceAccounts, setResourceAccounts] = useState<ResourceAccount[]>([]);

  // 表单数据
  const [formData, setFormData] = useState({
    model_id: '',
    channel_id: '',
    upstream_model_name: '',
    is_cost_priority: true,
    actual_cost_per_1k_input: '',
    actual_cost_per_1k_output: '',
    priority: '100',
    weight: '100',
    connect_timeout_ms: '2000',
    attempt_timeout_ms: '15000',
    stream_first_chunk_timeout_ms: '3000',
    failure_threshold: '5',
    recovery_timeout_seconds: '30',
    status: 'active' as ModelProvider['status'],
  });

  // 熔断操作数据
  const [circuitAction, setCircuitAction] = useState<'open' | 'close'>('open');
  const [circuitDuration, setCircuitDuration] = useState('30');
  const [circuitReason, setCircuitReason] = useState('');

  // 实例表单数据
  const [instanceFormData, setInstanceFormData] = useState({
    name: '',
    instance_type: 'api_key' as 'api_key' | 'resource_account' | 'session',
    resource_account_id: '',
    weight: '1',
    max_concurrency: '0',
    rpm_limit: '0',
    tpm_limit: '0',
    status: 'active' as 'active' | 'disabled' | 'cooling',
  });

  // 加载源头列表
  const loadProviders = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = {
        page: page.toString(),
        page_size: pageSize.toString(),
      };
      if (filterModelId && filterModelId !== 'all') params.model_id = filterModelId;
      if (filterChannelId && filterChannelId !== 'all') params.channel_id = filterChannelId;
      if (filterStatus && filterStatus !== 'all') params.status = filterStatus;
      if (filterCircuitState && filterCircuitState !== 'all') params.circuit_state = filterCircuitState;

      const response = await api.get('/api/admin/providers', { params });
      const data = response as { data?: { list?: ModelProvider[]; total?: number } };
      setProviders(data.data?.list || []);
      setTotal(data.data?.total || 0);
    } catch (error) {
      toast({
        title: '加载失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      });
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, filterModelId, filterChannelId, filterStatus, filterCircuitState, toast]);

  // 加载渠道列表
  const loadChannels = useCallback(async () => {
    try {
      const response = await api.get('/api/admin/channels', { params: { page_size: '100' } });
      const data = response as { data?: { list?: Channel[] } };
      setChannels(data.data?.list || []);
    } catch (error) {
      toast({
        title: '加载渠道失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      });
      setChannels([]);
    }
  }, [toast]);

  // 加载模型列表
  const loadModels = useCallback(async () => {
    try {
      const response = await api.get('/api/admin/models', { params: { page_size: '100' } });
      const data = response as { data?: { list?: Model[] } };
      setModels(data.data?.list || []);
    } catch (error) {
      toast({
        title: '加载模型失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      });
      setModels([]);
    }
  }, [toast]);

  useEffect(() => {
    loadProviders();
  }, [loadProviders]);

  useEffect(() => {
    loadChannels();
    loadModels();
  }, [loadChannels, loadModels]);

  // 加载资源账户列表
  const loadResourceAccounts = async () => {
    try {
      const response = await api.get('/api/admin/resource-accounts', { params: { page_size: '100' } });
      const data = response as { data?: { list?: ResourceAccount[] } };
      setResourceAccounts(data.data?.list || []);
    } catch (error) {
      toast({
        title: '加载资源账户失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      });
      setResourceAccounts([]);
    }
  };

  // 加载实例列表
  const loadInstances = useCallback(async (providerId: number, page: number = 1) => {
    setInstancesLoading(true);
    try {
      const response = await api.get(`/api/admin/providers/${providerId}/instances`, {
        params: { page, page_size: 20 }
      });
      const data = response as { data?: { list?: ProviderInstance[]; total?: number } };
      setInstances(data.data?.list || []);
      setInstancesTotal(data.data?.total || 0);
      setInstancesPage(page);
    } catch (error) {
      toast({
        title: '加载失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      });
    } finally {
      setInstancesLoading(false);
    }
  }, [toast]);

  // 打开实例管理模态框
  const openInstancesModal = (provider: ModelProvider) => {
    setSelectedProvider(provider);
    setShowInstancesModal(true);
    loadInstances(provider.id);
    loadResourceAccounts();
  };

  // 创建实例
  const handleCreateInstance = async () => {
    if (!selectedProvider) return;

    try {
      await api.post(`/api/admin/providers/${selectedProvider.id}/instances`, {
        name: instanceFormData.name,
        instance_type: instanceFormData.instance_type,
        resource_account_id: instanceFormData.resource_account_id ? parseInt(instanceFormData.resource_account_id) : undefined,
        weight: parseInt(instanceFormData.weight),
        max_concurrency: parseInt(instanceFormData.max_concurrency),
        rpm_limit: parseInt(instanceFormData.rpm_limit),
        tpm_limit: parseInt(instanceFormData.tpm_limit),
        status: instanceFormData.status,
      });
      toast({ title: '创建成功' });
      setShowCreateInstanceModal(false);
      resetInstanceForm();
      loadInstances(selectedProvider.id, instancesPage);
    } catch (error) {
      toast({ title: '创建失败', description: getErrorMessage(error), variant: 'destructive' });
    }
  };

  // 更新实例
  const handleUpdateInstance = async () => {
    if (!selectedInstance) return;

    try {
      await api.put(`/api/admin/provider-instances/${selectedInstance.id}`, {
        name: instanceFormData.name,
        instance_type: instanceFormData.instance_type,
        resource_account_id: instanceFormData.resource_account_id ? parseInt(instanceFormData.resource_account_id) : undefined,
        weight: parseInt(instanceFormData.weight),
        max_concurrency: parseInt(instanceFormData.max_concurrency),
        rpm_limit: parseInt(instanceFormData.rpm_limit),
        tpm_limit: parseInt(instanceFormData.tpm_limit),
        status: instanceFormData.status,
      });
      toast({ title: '更新成功' });
      setShowEditInstanceModal(false);
      setSelectedInstance(null);
      resetInstanceForm();
      if (selectedProvider) {
        loadInstances(selectedProvider.id, instancesPage);
      }
    } catch (error) {
      toast({ title: '更新失败', description: getErrorMessage(error), variant: 'destructive' });
    }
  };

  // 删除实例
  const handleDeleteInstance = async () => {
    if (!selectedInstance) return;

    try {
      await api.delete(`/api/admin/provider-instances/${selectedInstance.id}`);
      toast({ title: '删除成功' });
      setShowInstanceDeleteConfirm(false);
      setSelectedInstance(null);
      if (selectedProvider) {
        loadInstances(selectedProvider.id, instancesPage);
      }
    } catch (error) {
      toast({ title: '删除失败', description: getErrorMessage(error), variant: 'destructive' });
    }
  };

  // 启用/禁用实例
  const handleToggleInstanceStatus = async (instance: ProviderInstance) => {
    const action = instance.status === 'active' ? 'disable' : 'enable';
    try {
      await api.post(`/api/admin/provider-instances/${instance.id}/${action}`);
      toast({ title: action === 'enable' ? '已启用' : '已禁用' });
      if (selectedProvider) {
        loadInstances(selectedProvider.id, instancesPage);
      }
    } catch (error) {
      toast({ title: '操作失败', description: getErrorMessage(error), variant: 'destructive' });
    }
  };

  // 重置实例表单
  const resetInstanceForm = () => {
    setInstanceFormData({
      name: '',
      instance_type: 'api_key',
      resource_account_id: '',
      weight: '1',
      max_concurrency: '0',
      rpm_limit: '0',
      tpm_limit: '0',
      status: 'active',
    });
  };

  // 打开编辑实例模态框
  const openEditInstanceModal = (instance: ProviderInstance) => {
    setSelectedInstance(instance);
    setInstanceFormData({
      name: instance.name,
      instance_type: instance.instance_type,
      resource_account_id: instance.resource_account_id?.toString() || '',
      weight: instance.weight.toString(),
      max_concurrency: instance.max_concurrency.toString(),
      rpm_limit: instance.rpm_limit.toString(),
      tpm_limit: instance.tpm_limit.toString(),
      status: instance.status,
    });
    setShowEditInstanceModal(true);
  };

  // 创建源头
  const handleCreate = async () => {
    try {
      if (!formData.model_id || !formData.channel_id) {
        toast({ title: '请先选择模型和渠道', variant: 'destructive' });
        return;
      }

      await api.post('/api/admin/providers', {
        model_id: formData.model_id,
        channel_id: parseInt(formData.channel_id),
        upstream_model_name: formData.upstream_model_name || formData.model_id,
        actual_cost_per_1k_input: parseFloat(formData.actual_cost_per_1k_input),
        actual_cost_per_1k_output: parseFloat(formData.actual_cost_per_1k_output),
        is_cost_priority: formData.is_cost_priority,
        priority: parseInt(formData.priority),
        weight: parseInt(formData.weight),
        connect_timeout_ms: parseInt(formData.connect_timeout_ms),
        attempt_timeout_ms: parseInt(formData.attempt_timeout_ms),
        stream_first_chunk_timeout_ms: parseInt(formData.stream_first_chunk_timeout_ms),
        failure_threshold: parseInt(formData.failure_threshold),
        recovery_timeout_seconds: parseInt(formData.recovery_timeout_seconds),
        status: formData.status,
      });

      toast({ title: '创建成功' });
      setShowCreateModal(false);
      resetForm();
      loadProviders();
    } catch (error) {
      toast({ title: '创建失败', description: getErrorMessage(error), variant: 'destructive' });
    }
  };

  // 更新源头
  const handleUpdate = async () => {
    if (!selectedProvider) return;

    try {
      await api.put(`/api/admin/providers/${selectedProvider.id}`, {
        upstream_model_name: formData.upstream_model_name || undefined,
        actual_cost_per_1k_input: parseFloat(formData.actual_cost_per_1k_input),
        actual_cost_per_1k_output: parseFloat(formData.actual_cost_per_1k_output),
        is_cost_priority: formData.is_cost_priority,
        priority: parseInt(formData.priority),
        weight: parseInt(formData.weight),
        connect_timeout_ms: parseInt(formData.connect_timeout_ms),
        attempt_timeout_ms: parseInt(formData.attempt_timeout_ms),
        stream_first_chunk_timeout_ms: parseInt(formData.stream_first_chunk_timeout_ms),
        failure_threshold: parseInt(formData.failure_threshold),
        recovery_timeout_seconds: parseInt(formData.recovery_timeout_seconds),
        status: formData.status,
      });

      toast({ title: '更新成功' });
      setShowEditModal(false);
      setSelectedProvider(null);
      resetForm();
      loadProviders();
    } catch (error) {
      toast({ title: '更新失败', description: getErrorMessage(error), variant: 'destructive' });
    }
  };

  // 删除源头
  const handleDelete = async () => {
    if (!selectedProvider) return;

    try {
      await api.delete(`/api/admin/providers/${selectedProvider.id}`);
      toast({ title: '删除成功' });
      setShowDeleteConfirm(false);
      setSelectedProvider(null);
      loadProviders();
    } catch (error) {
      toast({ title: '删除失败', description: getErrorMessage(error), variant: 'destructive' });
    }
  };

  // 熔断操作
  const handleCircuitAction = async () => {
    if (!selectedProvider) return;

    try {
      await api.post(`/api/admin/providers/${selectedProvider.id}/circuit`, {
        action: circuitAction,
        duration: circuitAction === 'open' ? parseInt(circuitDuration) : undefined,
        reason: circuitReason,
      });

      toast({ title: circuitAction === 'open' ? '熔断器已打开' : '熔断器已关闭' });
      setShowCircuitModal(false);
      setSelectedProvider(null);
      setCircuitReason('');
      loadProviders();
    } catch (error) {
      toast({ title: '操作失败', description: getErrorMessage(error), variant: 'destructive' });
    }
  };

  // 启用/禁用源头
  const handleToggleStatus = async (provider: ModelProvider) => {
    const action = provider.status === 'active' ? 'disable' : 'enable';
    try {
      await api.post(`/api/admin/providers/${provider.id}/${action}`);
      toast({ title: action === 'enable' ? '已启用' : '已禁用' });
      loadProviders();
    } catch (error) {
      toast({ title: '操作失败', description: getErrorMessage(error), variant: 'destructive' });
    }
  };

  // 重置表单
  const resetForm = () => {
    setFormData({
      model_id: '',
      channel_id: '',
      upstream_model_name: '',
      is_cost_priority: true,
      actual_cost_per_1k_input: '',
      actual_cost_per_1k_output: '',
      priority: '100',
      weight: '100',
      connect_timeout_ms: '2000',
      attempt_timeout_ms: '15000',
      stream_first_chunk_timeout_ms: '3000',
      failure_threshold: '5',
      recovery_timeout_seconds: '30',
      status: 'active',
    });
  };

  // 打开编辑模态框
  const openEditModal = (provider: ModelProvider) => {
    setSelectedProvider(provider);
    setFormData({
      model_id: provider.model_id,
      channel_id: provider.channel_id.toString(),
      upstream_model_name: provider.upstream_model_name || provider.model_id,
      is_cost_priority: provider.is_cost_priority ?? true,
      actual_cost_per_1k_input: provider.actual_cost_per_1k_input.toString(),
      actual_cost_per_1k_output: provider.actual_cost_per_1k_output.toString(),
      priority: provider.priority.toString(),
      weight: provider.weight.toString(),
      connect_timeout_ms: String(provider.connect_timeout_ms ?? 2000),
      attempt_timeout_ms: String(provider.attempt_timeout_ms ?? 15000),
      stream_first_chunk_timeout_ms: String(provider.stream_first_chunk_timeout_ms ?? 3000),
      failure_threshold: provider.failure_threshold.toString(),
      recovery_timeout_seconds: String(provider.recovery_timeout_seconds ?? 30),
      status: provider.status,
    });
    setShowEditModal(true);
  };

  // 打开熔断模态框
  const openCircuitModal = (provider: ModelProvider, action: 'open' | 'close') => {
    setSelectedProvider(provider);
    setCircuitAction(action);
    setShowCircuitModal(true);
  };

  const totalPages = Math.ceil(total / pageSize);

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">模型源头管理</h1>
          <p className="text-gray-500 mt-1">管理模型与渠道的映射关系，配置成本和熔断策略</p>
        </div>
        <Button onClick={() => setShowCreateModal(true)}>
          新建源头
        </Button>
      </div>

      {/* 筛选条件 */}
      <Card className="p-4">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div>
            <Label>模型</Label>
            <Select value={filterModelId} onValueChange={setFilterModelId}>
              <SelectTrigger>
                <SelectValue placeholder="全部模型" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部模型</SelectItem>
                {models.map((model) => (
                  <SelectItem key={model.id} value={model.id}>
                    {model.name || model.id}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label>渠道</Label>
            <Select value={filterChannelId} onValueChange={setFilterChannelId}>
              <SelectTrigger>
                <SelectValue placeholder="全部渠道" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部渠道</SelectItem>
                {channels.map((channel) => (
                  <SelectItem key={channel.id} value={channel.id.toString()}>
                    {channel.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label>状态</Label>
            <Select value={filterStatus} onValueChange={setFilterStatus}>
              <SelectTrigger>
                <SelectValue placeholder="全部状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部状态</SelectItem>
                <SelectItem value="active">启用</SelectItem>
                <SelectItem value="disabled">禁用</SelectItem>
                <SelectItem value="cooling">冷却</SelectItem>
                <SelectItem value="circuit_open">熔断</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label>熔断状态</Label>
            <Select value={filterCircuitState} onValueChange={setFilterCircuitState}>
              <SelectTrigger>
                <SelectValue placeholder="全部" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部</SelectItem>
                <SelectItem value="closed">正常</SelectItem>
                <SelectItem value="open">熔断</SelectItem>
                <SelectItem value="half_open">半开</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      </Card>

      {/* 源头列表 */}
      <Card>
        {loading ? (
          <Loading />
        ) : providers.length === 0 ? (
          <EmptyState
            title="暂无源头"
            description="点击上方按钮创建第一个模型源头"
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">模型</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">渠道</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">成本(输入/输出)</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">优先级</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">状态</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">熔断</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">健康分</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">请求统计</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {providers.map((provider) => (
                  <tr key={provider.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3">
                      <div className="font-medium">{provider.model_name || provider.model_id}</div>
                      <div className="text-xs text-gray-500">{provider.model_id}</div>
                    </td>
                    <td className="px-4 py-3">
                      <div>{provider.channel_name}</div>
                      <div className="text-xs text-gray-500">ID: {provider.channel_id}</div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="text-sm">
                        <span className="text-blue-600">${provider.actual_cost_per_1k_input}</span>
                        {' / '}
                        <span className="text-green-600">${provider.actual_cost_per_1k_output}</span>
                      </div>
                      <div className="text-xs text-gray-500">每1K tokens</div>
                    </td>
                    <td className="px-4 py-3">
                      <div>{provider.priority}</div>
                      <div className="text-xs text-gray-500">权重: {provider.weight}</div>
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={provider.status} />
                    </td>
                    <td className="px-4 py-3">
                      <CircuitBadge state={provider.circuit_state} />
                      {provider.circuit_state !== 'closed' && (
                        <div className="text-xs text-gray-500 mt-1">
                          失败: {provider.failure_count}/{provider.failure_threshold}
                        </div>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <HealthScore score={provider.health_score} />
                    </td>
                    <td className="px-4 py-3">
                      <div className="text-sm">
                        总计: {provider.total_requests}
                      </div>
                      <div className="text-xs">
                        <span className="text-green-600">成功: {provider.success_requests}</span>
                        {' / '}
                        <span className="text-red-600">失败: {provider.failed_requests}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => openEditModal(provider)}
                        >
                          编辑
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => openInstancesModal(provider)}
                        >
                          实例
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleToggleStatus(provider)}
                        >
                          {provider.status === 'active' ? '禁用' : '启用'}
                        </Button>
                        {provider.circuit_state === 'closed' ? (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => openCircuitModal(provider, 'open')}
                          >
                            熔断
                          </Button>
                        ) : (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => openCircuitModal(provider, 'close')}
                          >
                            恢复
                          </Button>
                        )}
                        <Button
                          variant="destructive"
                          size="sm"
                          onClick={() => {
                            setSelectedProvider(provider);
                            setShowDeleteConfirm(true);
                          }}
                        >
                          删除
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* 分页 */}
        {totalPages > 1 && (
          <div className="p-4 border-t">
            <Pagination
              currentPage={page}
              totalPages={totalPages}
              total={total}
              pageSize={pageSize}
              onPageChange={setPage}
            />
          </div>
        )}
      </Card>

      {/* 创建模态框 */}
      <Modal
        isOpen={showCreateModal}
        onClose={() => {
          setShowCreateModal(false);
          resetForm();
        }}
        title="新建模型源头"
        size="lg"
      >
        <div className="space-y-4">
          <div>
            <Label>模型 *</Label>
            <Select value={formData.model_id} onValueChange={(value) => setFormData({ ...formData, model_id: value })}>
              <SelectTrigger>
                <SelectValue placeholder="选择模型" />
              </SelectTrigger>
              <SelectContent>
                {models.map((model) => (
                  <SelectItem key={model.id} value={model.id}>
                    {model.name || model.id}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label>渠道 *</Label>
            <Select value={formData.channel_id} onValueChange={(value) => setFormData({ ...formData, channel_id: value })}>
              <SelectTrigger>
                <SelectValue placeholder="选择渠道" />
              </SelectTrigger>
              <SelectContent>
                {channels.map((channel) => (
                  <SelectItem key={channel.id} value={channel.id.toString()}>
                    {channel.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>输入成本 ($/1K tokens) *</Label>
              <Input
                type="number"
                step="0.000001"
                value={formData.actual_cost_per_1k_input}
                onChange={(e) => setFormData({ ...formData, actual_cost_per_1k_input: e.target.value })}
                placeholder="0.001"
              />
            </div>
            <div>
              <Label>输出成本 ($/1K tokens) *</Label>
              <Input
                type="number"
                step="0.000001"
                value={formData.actual_cost_per_1k_output}
                onChange={(e) => setFormData({ ...formData, actual_cost_per_1k_output: e.target.value })}
                placeholder="0.002"
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>优先级</Label>
              <Input
                type="number"
                value={formData.priority}
                onChange={(e) => setFormData({ ...formData, priority: e.target.value })}
              />
            </div>
            <div>
              <Label>权重</Label>
              <Input
                type="number"
                value={formData.weight}
                onChange={(e) => setFormData({ ...formData, weight: e.target.value })}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>失败阈值</Label>
              <Input
                type="number"
                value={formData.failure_threshold}
                onChange={(e) => setFormData({ ...formData, failure_threshold: e.target.value })}
              />
            </div>
            <div>
              <Label>恢复超时 (秒)</Label>
              <Input
                type="number"
                value={formData.recovery_timeout_seconds}
                onChange={(e) => setFormData({ ...formData, recovery_timeout_seconds: e.target.value })}
              />
            </div>
          </div>
          <div>
            <Label>状态</Label>
            <Select
              value={formData.status}
              onValueChange={(value) => setFormData({ ...formData, status: value as ModelProvider['status'] })}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="active">启用</SelectItem>
                <SelectItem value="disabled">禁用</SelectItem>
                <SelectItem value="cooling">冷却</SelectItem>
                <SelectItem value="circuit_open" disabled>
                  熔断
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex justify-end gap-2 pt-4">
            <Button variant="outline" onClick={() => {
              setShowCreateModal(false);
              resetForm();
            }}>
              取消
            </Button>
            <Button onClick={handleCreate}>
              创建
            </Button>
          </div>
        </div>
      </Modal>

      {/* 编辑模态框 */}
      <Modal
        isOpen={showEditModal}
        onClose={() => {
          setShowEditModal(false);
          setSelectedProvider(null);
          resetForm();
        }}
        title="编辑模型源头"
        size="lg"
      >
        <div className="space-y-4">
          <div>
            <Label>模型</Label>
            <Input value={formData.model_id} disabled />
          </div>
          <div>
            <Label>渠道</Label>
            <Input value={channels.find(c => c.id.toString() === formData.channel_id)?.name || ''} disabled />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>输入成本 ($/1K tokens)</Label>
              <Input
                type="number"
                step="0.000001"
                value={formData.actual_cost_per_1k_input}
                onChange={(e) => setFormData({ ...formData, actual_cost_per_1k_input: e.target.value })}
              />
            </div>
            <div>
              <Label>输出成本 ($/1K tokens)</Label>
              <Input
                type="number"
                step="0.000001"
                value={formData.actual_cost_per_1k_output}
                onChange={(e) => setFormData({ ...formData, actual_cost_per_1k_output: e.target.value })}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>优先级</Label>
              <Input
                type="number"
                value={formData.priority}
                onChange={(e) => setFormData({ ...formData, priority: e.target.value })}
              />
            </div>
            <div>
              <Label>权重</Label>
              <Input
                type="number"
                value={formData.weight}
                onChange={(e) => setFormData({ ...formData, weight: e.target.value })}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>失败阈值</Label>
              <Input
                type="number"
                value={formData.failure_threshold}
                onChange={(e) => setFormData({ ...formData, failure_threshold: e.target.value })}
              />
            </div>
            <div>
              <Label>恢复超时 (秒)</Label>
              <Input
                type="number"
                value={formData.recovery_timeout_seconds}
                onChange={(e) => setFormData({ ...formData, recovery_timeout_seconds: e.target.value })}
              />
            </div>
          </div>
          <div>
            <Label>状态</Label>
            <Select
              value={formData.status}
              onValueChange={(value) => setFormData({ ...formData, status: value as ModelProvider['status'] })}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="active">启用</SelectItem>
                <SelectItem value="disabled">禁用</SelectItem>
                <SelectItem value="cooling">冷却</SelectItem>
                <SelectItem value="circuit_open" disabled>
                  熔断
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex justify-end gap-2 pt-4">
            <Button variant="outline" onClick={() => {
              setShowEditModal(false);
              setSelectedProvider(null);
              resetForm();
            }}>
              取消
            </Button>
            <Button onClick={handleUpdate}>
              保存
            </Button>
          </div>
        </div>
      </Modal>

      {/* 熔断操作模态框 */}
      <Modal
        isOpen={showCircuitModal}
        onClose={() => {
          setShowCircuitModal(false);
          setSelectedProvider(null);
          setCircuitReason('');
        }}
        title={circuitAction === 'open' ? '手动熔断' : '恢复熔断器'}
      >
        <div className="space-y-4">
          <p className="text-gray-600">
            {circuitAction === 'open'
              ? `确定要手动熔断源头 "${selectedProvider?.model_name}" (渠道: ${selectedProvider?.channel_name}) 吗？`
              : `确定要恢复源头 "${selectedProvider?.model_name}" (渠道: ${selectedProvider?.channel_name}) 的熔断器吗？`
            }
          </p>
          {circuitAction === 'open' && (
            <div>
              <Label>熔断时长 (秒)</Label>
              <Input
                type="number"
                value={circuitDuration}
                onChange={(e) => setCircuitDuration(e.target.value)}
                placeholder="30"
              />
            </div>
          )}
          <div>
            <Label>原因 (可选)</Label>
            <Input
              value={circuitReason}
              onChange={(e) => setCircuitReason(e.target.value)}
              placeholder="请输入操作原因"
            />
          </div>
          <div className="flex justify-end gap-2 pt-4">
            <Button variant="outline" onClick={() => {
              setShowCircuitModal(false);
              setSelectedProvider(null);
              setCircuitReason('');
            }}>
              取消
            </Button>
            <Button
              variant={circuitAction === 'open' ? 'destructive' : 'default'}
              onClick={handleCircuitAction}
            >
              {circuitAction === 'open' ? '确认熔断' : '确认恢复'}
            </Button>
          </div>
        </div>
      </Modal>

      {/* 删除确认对话框 */}
      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={(open) => {
          if (!open) {
            setShowDeleteConfirm(false);
            setSelectedProvider(null);
          }
        }}
        onConfirm={handleDelete}
        title="删除源头"
        description={`确定要删除源头 "${selectedProvider?.model_name}" (渠道: ${selectedProvider?.channel_name}) 吗？此操作不可撤销。`}
        confirmText="删除"
        cancelText="取消"
        variant="destructive"
      />

      {/* 实例管理模态框 */}
      <Modal
        isOpen={showInstancesModal}
        onClose={() => {
          setShowInstancesModal(false);
          setSelectedProvider(null);
          setInstances([]);
        }}
        title={`实例管理 - ${selectedProvider?.model_name}`}
        size="xl"
      >
        <div className="space-y-4">
          <div className="flex justify-between items-center">
            <div className="text-sm text-gray-500">
              渠道: {selectedProvider?.channel_name}
            </div>
            <Button onClick={() => setShowCreateInstanceModal(true)}>
              新建实例
            </Button>
          </div>

          {instancesLoading ? (
            <Loading />
          ) : instances.length === 0 ? (
            <EmptyState
              title="暂无实例"
              description="点击上方按钮创建第一个实例"
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">名称</th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">类型</th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">资源账户</th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">权重</th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">状态</th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">限流</th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">统计</th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {instances.map((instance) => (
                    <tr key={instance.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3">
                        <div className="font-medium">{instance.name}</div>
                        <div className="text-xs text-gray-500">ID: {instance.id}</div>
                      </td>
                      <td className="px-4 py-3">
                        <span className="px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                          {instance.instance_type === 'api_key' ? 'API密钥' :
                           instance.instance_type === 'resource_account' ? '资源账户' : '会话'}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        {instance.account_name || '-'}
                      </td>
                      <td className="px-4 py-3">
                        {instance.weight}
                      </td>
                      <td className="px-4 py-3">
                        <StatusBadge status={instance.status} />
                      </td>
                      <td className="px-4 py-3">
                        <div className="text-sm">
                          并发: {instance.max_concurrency || '∞'}
                        </div>
                        <div className="text-xs text-gray-500">
                          RPM: {instance.rpm_limit || '∞'} / TPM: {instance.tpm_limit || '∞'}
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <div className="text-sm">
                          总计: {instance.total_requests}
                        </div>
                        <div className="text-xs">
                          <span className="text-green-600">成功率: {Number(instance.success_rate).toFixed(1)}%</span>
                          {' / '}
                          <span className="text-gray-600">延迟: {instance.avg_latency_ms}ms</span>
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => openEditInstanceModal(instance)}
                          >
                            编辑
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleToggleInstanceStatus(instance)}
                          >
                            {instance.status === 'active' ? '禁用' : '启用'}
                          </Button>
                          <Button
                            variant="destructive"
                            size="sm"
                            onClick={() => {
                              setSelectedInstance(instance);
                              setShowInstanceDeleteConfirm(true);
                            }}
                          >
                            删除
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {/* 实例分页 */}
          {instancesTotal > 20 && (
            <div className="flex justify-center pt-4">
              <Pagination
                currentPage={instancesPage}
                totalPages={Math.ceil(instancesTotal / 20)}
                total={instancesTotal}
                pageSize={20}
                onPageChange={(page) => {
                  if (selectedProvider) {
                    loadInstances(selectedProvider.id, page);
                  }
                }}
              />
            </div>
          )}
        </div>
      </Modal>

      {/* 创建实例模态框 */}
      <Modal
        isOpen={showCreateInstanceModal}
        onClose={() => {
          setShowCreateInstanceModal(false);
          resetInstanceForm();
        }}
        title="新建实例"
        size="lg"
      >
        <div className="space-y-4">
          <div>
            <Label>实例名称 *</Label>
            <Input
              value={instanceFormData.name}
              onChange={(e) => setInstanceFormData({ ...instanceFormData, name: e.target.value })}
              placeholder="实例名称"
            />
          </div>
          <div>
            <Label>实例类型 *</Label>
            <Select
              value={instanceFormData.instance_type}
              onValueChange={(value) =>
                setInstanceFormData({ ...instanceFormData, instance_type: value as typeof instanceFormData.instance_type })
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="api_key">API密钥</SelectItem>
                <SelectItem value="resource_account">资源账户</SelectItem>
                <SelectItem value="session">会话</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {instanceFormData.instance_type === 'resource_account' && (
            <div>
              <Label>资源账户</Label>
              <Select
                value={instanceFormData.resource_account_id}
                onValueChange={(value) => setInstanceFormData({ ...instanceFormData, resource_account_id: value })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择资源账户" />
                </SelectTrigger>
                <SelectContent>
                  {resourceAccounts.map((account) => (
                    <SelectItem key={account.id} value={account.id.toString()}>
                      {account.account_name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>权重</Label>
              <Input
                type="number"
                value={instanceFormData.weight}
                onChange={(e) => setInstanceFormData({ ...instanceFormData, weight: e.target.value })}
                placeholder="1"
              />
            </div>
            <div>
              <Label>最大并发数 (0=不限制)</Label>
              <Input
                type="number"
                value={instanceFormData.max_concurrency}
                onChange={(e) => setInstanceFormData({ ...instanceFormData, max_concurrency: e.target.value })}
                placeholder="0"
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>RPM限制 (0=不限制)</Label>
              <Input
                type="number"
                value={instanceFormData.rpm_limit}
                onChange={(e) => setInstanceFormData({ ...instanceFormData, rpm_limit: e.target.value })}
                placeholder="0"
              />
            </div>
            <div>
              <Label>TPM限制 (0=不限制)</Label>
              <Input
                type="number"
                value={instanceFormData.tpm_limit}
                onChange={(e) => setInstanceFormData({ ...instanceFormData, tpm_limit: e.target.value })}
                placeholder="0"
              />
            </div>
          </div>
          <div>
            <Label>状态</Label>
            <Select
              value={instanceFormData.status}
              onValueChange={(value) =>
                setInstanceFormData({ ...instanceFormData, status: value as typeof instanceFormData.status })
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="active">启用</SelectItem>
                <SelectItem value="disabled">禁用</SelectItem>
                <SelectItem value="cooling">冷却</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex justify-end gap-2 pt-4">
            <Button variant="outline" onClick={() => {
              setShowCreateInstanceModal(false);
              resetInstanceForm();
            }}>
              取消
            </Button>
            <Button onClick={handleCreateInstance}>
              创建
            </Button>
          </div>
        </div>
      </Modal>

      {/* 编辑实例模态框 */}
      <Modal
        isOpen={showEditInstanceModal}
        onClose={() => {
          setShowEditInstanceModal(false);
          setSelectedInstance(null);
          resetInstanceForm();
        }}
        title="编辑实例"
        size="lg"
      >
        <div className="space-y-4">
          <div>
            <Label>实例名称</Label>
            <Input
              value={instanceFormData.name}
              onChange={(e) => setInstanceFormData({ ...instanceFormData, name: e.target.value })}
            />
          </div>
          <div>
            <Label>实例类型</Label>
            <Select
              value={instanceFormData.instance_type}
              onValueChange={(value) =>
                setInstanceFormData({ ...instanceFormData, instance_type: value as typeof instanceFormData.instance_type })
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="api_key">API密钥</SelectItem>
                <SelectItem value="resource_account">资源账户</SelectItem>
                <SelectItem value="session">会话</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {instanceFormData.instance_type === 'resource_account' && (
            <div>
              <Label>资源账户</Label>
              <Select
                value={instanceFormData.resource_account_id}
                onValueChange={(value) => setInstanceFormData({ ...instanceFormData, resource_account_id: value })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择资源账户" />
                </SelectTrigger>
                <SelectContent>
                  {resourceAccounts.map((account) => (
                    <SelectItem key={account.id} value={account.id.toString()}>
                      {account.account_name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>权重</Label>
              <Input
                type="number"
                value={instanceFormData.weight}
                onChange={(e) => setInstanceFormData({ ...instanceFormData, weight: e.target.value })}
              />
            </div>
            <div>
              <Label>最大并发数 (0=不限制)</Label>
              <Input
                type="number"
                value={instanceFormData.max_concurrency}
                onChange={(e) => setInstanceFormData({ ...instanceFormData, max_concurrency: e.target.value })}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>RPM限制 (0=不限制)</Label>
              <Input
                type="number"
                value={instanceFormData.rpm_limit}
                onChange={(e) => setInstanceFormData({ ...instanceFormData, rpm_limit: e.target.value })}
              />
            </div>
            <div>
              <Label>TPM限制 (0=不限制)</Label>
              <Input
                type="number"
                value={instanceFormData.tpm_limit}
                onChange={(e) => setInstanceFormData({ ...instanceFormData, tpm_limit: e.target.value })}
              />
            </div>
          </div>
          <div>
            <Label>状态</Label>
            <Select
              value={instanceFormData.status}
              onValueChange={(value) =>
                setInstanceFormData({ ...instanceFormData, status: value as typeof instanceFormData.status })
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="active">启用</SelectItem>
                <SelectItem value="disabled">禁用</SelectItem>
                <SelectItem value="cooling">冷却</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex justify-end gap-2 pt-4">
            <Button variant="outline" onClick={() => {
              setShowEditInstanceModal(false);
              setSelectedInstance(null);
              resetInstanceForm();
            }}>
              取消
            </Button>
            <Button onClick={handleUpdateInstance}>
              保存
            </Button>
          </div>
        </div>
      </Modal>

      {/* 删除实例确认对话框 */}
      <ConfirmDialog
        open={showInstanceDeleteConfirm}
        onOpenChange={(open) => {
          if (!open) {
            setShowInstanceDeleteConfirm(false);
            setSelectedInstance(null);
          }
        }}
        onConfirm={handleDeleteInstance}
        title="删除实例"
        description={`确定要删除实例 "${selectedInstance?.name}" 吗？此操作不可撤销。`}
        confirmText="删除"
        cancelText="取消"
        variant="destructive"
      />
    </div>
  );
}
