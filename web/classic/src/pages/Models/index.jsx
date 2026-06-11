/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Avatar,
  Banner,
  Button,
  Card,
  Dropdown,
  Empty,
  Input,
  Layout,
  Modal,
  Select,
  SideSheet,
  Spin,
  SplitButtonGroup,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconClose,
  IconCopy,
  IconInfoCircle,
  IconRefresh,
  IconSearch,
  IconTreeTriangleDown,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, copy, showError, showInfo, showSuccess } from '../../helpers';
import { renderModelTag } from '../../helpers/render';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { MODEL_TABLE_PAGE_SIZE } from '../../constants';

const { Content } = Layout;
const { Paragraph, Text, Title } = Typography;

const normalizeModelDetail = (model) => {
  if (typeof model === 'string') {
    return {
      key: model,
      model_name: model,
      channels: [],
      mapped: false,
      mappings: [],
    };
  }
  const modelName = model?.model_name || model?.id || '';
  return {
    key: modelName,
    model_name: modelName,
    channels: Array.isArray(model?.channels) ? model.channels : [],
    mapped: Boolean(model?.mapped),
    mappings: Array.isArray(model?.mappings) ? model.mappings : [],
  };
};

const getChannelName = (channel) =>
  channel?.name || `#${channel?.id || channel?.channel_id}`;

const getMappingDisplayName = (mapping, modelName) => {
  if (!mapping) return '';
  if (mapping.source === modelName) return mapping.target;
  if (mapping.target === modelName) return mapping.source;
  return mapping.target || mapping.source || '';
};

const MODEL_TEST_STATUS = {
  untested: 0,
  available: 1,
  unavailable: 2,
};

const getModelTestStatusText = (status, t) => {
  if (status === MODEL_TEST_STATUS.available) return t('可用');
  if (status === MODEL_TEST_STATUS.unavailable) return t('不可用');
  return t('未测试');
};

const getModelTestStatusColor = (status) => {
  if (status === MODEL_TEST_STATUS.available) return 'green';
  if (status === MODEL_TEST_STATUS.unavailable) return 'red';
  return 'grey';
};

const appendResponseInfo = (message, response, t) => {
  if (!response) return message;
  return `${message}\n${t('返回信息')}：${response}`;
};

const getTestItems = (record) => {
  const channels = Array.isArray(record?.channels) ? record.channels : [];
  const mappings = Array.isArray(record?.mappings) ? record.mappings : [];
  const channelById = new Map(
    channels.map((channel) => [channel.id ?? channel.channel_id, channel]),
  );

  if (mappings.length > 0) {
    return mappings.map((mapping) => {
      const channel = channelById.get(mapping.channel_id) || {
        id: mapping.channel_id,
        name: mapping.channel_name || `#${mapping.channel_id}`,
      };
      return {
        key: `${mapping.channel_id}-${mapping.source}-${mapping.target}`,
        channel,
        channelId: mapping.channel_id,
        channelName: mapping.channel_name || getChannelName(channel),
        sourceModel: mapping.target || mapping.source,
        targetModel: mapping.source,
        displayModel: mapping.source || mapping.target,
        mapped: true,
        testStatus: mapping.test_status ?? 0,
        testTime: mapping.test_time ?? 0,
        responseTime: mapping.response_time ?? 0,
        testError: mapping.test_error || '',
        testResponse: mapping.test_response || '',
      };
    });
  }

  return channels.map((channel) => ({
    key: `${channel.id}-${record.model_name}`,
    channel,
    channelId: channel.id,
    channelName: getChannelName(channel),
    sourceModel: record.model_name,
    targetModel: record.model_name,
    displayModel: record.model_name,
    mapped: false,
    testStatus: channel.test_status ?? 0,
    testTime: channel.test_time ?? 0,
    responseTime: channel.response_time ?? 0,
    testError: channel.test_error || '',
    testResponse: channel.test_response || '',
  }));
};

const Models = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [models, setModels] = useState([]);
  const [loading, setLoading] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  const [mappingInputs, setMappingInputs] = useState({});
  const [savingMappingKeys, setSavingMappingKeys] = useState(new Set());
  const [editingModelRecord, setEditingModelRecord] = useState(null);
  const [testModalRecord, setTestModalRecord] = useState(null);
  const [modelSearchKeyword, setModelSearchKeyword] = useState('');
  const [modelTablePage, setModelTablePage] = useState(1);
  const [selectedEndpointType, setSelectedEndpointType] = useState('');
  const [isStreamTest, setIsStreamTest] = useState(false);
  const [selectedTestItemKeys, setSelectedTestItemKeys] = useState([]);
  const [isBatchTesting, setIsBatchTesting] = useState(false);
  const [modelTestResults, setModelTestResults] = useState({});
  const [testingItemKeys, setTestingItemKeys] = useState(new Set());
  const allSelectingRef = useRef(false);
  const batchTestingRef = useRef(false);

  const modelRows = useMemo(() => models.map(normalizeModelDetail), [models]);

  const filteredModels = useMemo(() => {
    const keyword = searchValue.trim().toLowerCase();
    return modelRows.filter(
      (model) => !keyword || model.model_name.toLowerCase().includes(keyword),
    );
  }, [modelRows, searchValue]);

  const currentEditingModel = useMemo(() => {
    if (!editingModelRecord) return null;
    return (
      modelRows.find((model) => model.model_name === editingModelRecord.model_name) ||
      editingModelRecord
    );
  }, [editingModelRecord, modelRows]);

  const testModalItems = useMemo(
    () => (testModalRecord ? getTestItems(testModalRecord) : []),
    [testModalRecord],
  );

  const filteredTestModalItems = useMemo(() => {
    const keyword = modelSearchKeyword.trim().toLowerCase();
    return testModalItems.filter(
      (item) =>
        !keyword ||
        item.sourceModel.toLowerCase().includes(keyword) ||
        item.channelName.toLowerCase().includes(keyword) ||
        item.displayModel.toLowerCase().includes(keyword),
    );
  }, [modelSearchKeyword, testModalItems]);

  const pagedTestModalItems = useMemo(() => {
    const start = (modelTablePage - 1) * MODEL_TABLE_PAGE_SIZE;
    return filteredTestModalItems.slice(start, start + MODEL_TABLE_PAGE_SIZE);
  }, [filteredTestModalItems, modelTablePage]);

  const selectedFilteredTestItems = useMemo(() => {
    if (selectedTestItemKeys.length === 0) return filteredTestModalItems;
    const selectedKeySet = new Set(selectedTestItemKeys);
    return filteredTestModalItems.filter((item) => selectedKeySet.has(item.key));
  }, [filteredTestModalItems, selectedTestItemKeys]);

  const fetchModels = async () => {
    setLoading(true);
    try {
      let res = await API.get('/api/channel/models_enabled_details');
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setModels(Array.isArray(data) ? data : []);
      setSelectedRowKeys([]);
    } catch (error) {
      try {
        const res = await API.get('/api/channel/models_enabled');
        const { success, message, data } = res.data;
        if (!success) {
          showError(message);
          return;
        }
        setModels(Array.isArray(data) ? [...data].sort() : []);
        setSelectedRowKeys([]);
      } catch (fallbackError) {
        showError(fallbackError.message || error.message || t('获取模型列表失败'));
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchModels();
  }, []);

  const copyModels = async (names) => {
    const list = Array.isArray(names) ? names : [names];
    if (list.length === 0) return;
    const ok = await copy(list.join('\n'));
    if (ok) {
      showSuccess(t('已复制到剪贴板'));
    }
  };

  const getEditRows = (record) => {
    const channels = Array.isArray(record?.channels) ? record.channels : [];
    const mappings = Array.isArray(record?.mappings) ? record.mappings : [];
    const normalizeChannelId = (channelId) => String(channelId ?? '');
    const mappingByChannelId = new Map(
      mappings
        .filter((mapping) => mapping.target === record.model_name)
        .map((mapping) => [normalizeChannelId(mapping.channel_id), mapping]),
    );
    const channelIds = new Set(
      channels.map((channel) => normalizeChannelId(channel.id ?? channel.channel_id)),
    );

    const channelRows = channels.map((channel) => {
      const channelId = channel.id ?? channel.channel_id;
      const mapping = mappingByChannelId.get(normalizeChannelId(channelId));
      return {
        key: `${channelId}-${record.model_name}`,
        channelId,
        channelName: mapping?.channel_name || getChannelName(channel),
        sourceModel: record.model_name,
        targetModel: mapping?.source || '',
        originalSourceModel: mapping?.source || '',
        testStatus: channel.test_status ?? 0,
        testTime: channel.test_time ?? 0,
        responseTime: channel.response_time ?? 0,
        testError: channel.test_error || '',
        testResponse: channel.test_response || '',
      };
    });

    const orphanMappingRows = mappings
      .filter((mapping) => !channelIds.has(normalizeChannelId(mapping.channel_id)))
      .map((mapping) => ({
        key: `${mapping.channel_id}-${mapping.target || mapping.source}`,
        channelId: mapping.channel_id,
        channelName: mapping.channel_name || `#${mapping.channel_id}`,
        sourceModel: mapping.target || mapping.source,
        targetModel: mapping.source,
        originalSourceModel: mapping.source,
        testStatus: mapping.test_status ?? 0,
        testTime: mapping.test_time ?? 0,
        responseTime: mapping.response_time ?? 0,
        testError: mapping.test_error || '',
        testResponse: mapping.test_response || '',
      }));

    return [...channelRows, ...orphanMappingRows];
  };

  const getMappingInputKey = (row) => `${row.channelId}-${row.sourceModel}`;

  const setMappingValue = (row, value) => {
    setMappingInputs((prev) => ({
      ...prev,
      [getMappingInputKey(row)]: value,
    }));
  };

  const parseModelMapping = (modelMapping) => {
    if (!modelMapping) return {};
    try {
      const parsed = JSON.parse(modelMapping);
      return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
        ? parsed
        : {};
    } catch (error) {
      showError(t('JSON解析错误:') + error.message);
      return null;
    }
  };

  const saveMapping = async (row) => {
    const inputKey = getMappingInputKey(row);
    const nextUpstreamModel = (mappingInputs[inputKey] ?? row.targetModel ?? '').trim();
    setSavingMappingKeys((prev) => new Set([...prev, inputKey]));
    try {
      const channelRes = await API.get(`/api/channel/${row.channelId}`);
      const { success, message, data } = channelRes.data;
      if (!success) {
        showError(message);
        return;
      }

      const modelMapping = parseModelMapping(data?.model_mapping);
      if (modelMapping === null) return;
      if (row.originalSourceModel) {
        delete modelMapping[row.originalSourceModel];
      }
      if (nextUpstreamModel) {
        modelMapping[nextUpstreamModel] = row.sourceModel;
      }

      const updateRes = await API.put('/api/channel/', {
        id: row.channelId,
        model_mapping: Object.keys(modelMapping).length
          ? JSON.stringify(modelMapping, null, 2)
          : '',
      });
      if (updateRes.data.success) {
        showSuccess(t('保存成功'));
        setMappingInputs((prev) => {
          const next = { ...prev };
          delete next[inputKey];
          return next;
        });
        await fetchModels();
      } else {
        showError(updateRes.data.message);
      }
    } catch (error) {
      showError(error.message || t('保存失败'));
    } finally {
      setSavingMappingKeys((prev) => {
        const next = new Set(prev);
        next.delete(inputKey);
        return next;
      });
    }
  };

  const endpointTypeOptions = [
    { value: '', label: t('自动检测') },
    { value: 'openai', label: 'OpenAI (/v1/chat/completions)' },
    { value: 'openai-response', label: 'OpenAI Response (/v1/responses)' },
    {
      value: 'openai-response-compact',
      label: 'OpenAI Response Compaction (/v1/responses/compact)',
    },
    { value: 'anthropic', label: 'Anthropic (/v1/messages)' },
    { value: 'gemini', label: 'Gemini (/v1beta/models/{model}:generateContent)' },
    { value: 'jina-rerank', label: 'Jina Rerank (/v1/rerank)' },
    { value: 'image-generation', label: t('图像生成') + ' (/v1/images/generations)' },
    { value: 'embeddings', label: 'Embeddings (/v1/embeddings)' },
  ];

  const streamToggleDisabled = [
    'embeddings',
    'image-generation',
    'jina-rerank',
    'openai-response-compact',
  ].includes(selectedEndpointType);

  useEffect(() => {
    if (streamToggleDisabled && isStreamTest) {
      setIsStreamTest(false);
    }
  }, [streamToggleDisabled, isStreamTest]);

  const testItem = async (
    item,
    endpointType = selectedEndpointType,
    stream = isStreamTest,
  ) => {
    const testKey = item.key;
    setTestingItemKeys((prev) => new Set([...prev, testKey]));
    try {
      const params = new URLSearchParams({ model: item.sourceModel });
      if (endpointType) params.set('endpoint_type', endpointType);
      if (stream) params.set('stream', 'true');
      const url = `/api/channel/test/${item.channelId}?${params.toString()}`;
      const res = await API.get(url);
      const { success, message, time, error_code, response } = res.data;
      setModelTestResults((prev) => ({
        ...prev,
        [testKey]: {
          success,
          message,
          time: time || 0,
          errorCode: error_code || null,
          response: response || '',
        },
      }));
      if (success) {
        showInfo(
          appendResponseInfo(
            t('通道 ${name} 测试成功，模型 ${model} 耗时 ${time.toFixed(2)} 秒。')
              .replace('${name}', item.channelName)
              .replace('${model}', item.sourceModel)
              .replace('${time.toFixed(2)}', (time || 0).toFixed(2)),
            response,
            t,
          ),
        );
      } else {
        showError(message);
      }
    } catch (error) {
      setModelTestResults((prev) => ({
        ...prev,
        [testKey]: {
          success: false,
          message: error.message || t('网络错误'),
          time: 0,
          errorCode: null,
          response: '',
        },
      }));
      showError(error.message || t('测试失败'));
    } finally {
      setTestingItemKeys((prev) => {
        const next = new Set(prev);
        next.delete(testKey);
        return next;
      });
    }
  };

  const batchTestItems = async (items) => {
    if (items.length === 0) {
      showError(t('当前模型暂无可测试渠道'));
      return;
    }
    batchTestingRef.current = true;
    setIsBatchTesting(true);
    for (const item of items) {
      if (!batchTestingRef.current) break;
      await testItem(item, selectedEndpointType, isStreamTest);
    }
    batchTestingRef.current = false;
    setIsBatchTesting(false);
    await fetchModels();
  };

  const closeTestModal = () => {
    if (isBatchTesting) {
      batchTestingRef.current = false;
      setIsBatchTesting(false);
      showInfo(t('已停止批量测试'));
    }
    setTestModalRecord(null);
  };

  const batchTestRecord = async (record) => {
    const items = getTestItems(record);
    await batchTestItems(items);
  };

  const batchTestSelectedModels = () => {
    if (selectedRowKeys.length === 0) {
      showError(t('请先选择模型！'));
      return;
    }
    const selectedKeySet = new Set(selectedRowKeys);
    const selectedModels = filteredModels.filter((model) =>
      selectedKeySet.has(model.model_name),
    );
    const items = selectedModels.flatMap((model) => getTestItems(model));
    if (items.length === 0) {
      showError(t('当前模型暂无可测试渠道'));
      return;
    }
    Modal.confirm({
      title: t('测试所选模型'),
      content: t('确定要测试选中的 ${count} 个模型吗？').replace(
        '${count}',
        selectedModels.length,
      ),
      onOk: () => batchTestItems(items),
    });
  };

  const openTestOptions = (record) => {
    const items = getTestItems(record);
    if (items.length === 0) {
      showError(t('当前模型暂无可测试渠道'));
      return;
    }
    setModelSearchKeyword('');
    setModelTablePage(1);
    setSelectedEndpointType('');
    setIsStreamTest(false);
    setSelectedTestItemKeys([]);
    setTestModalRecord(record);
  };

  const openEditModel = (record) => setEditingModelRecord(record);

  const renderMappingColumn = (record) => {
    const mappings = Array.isArray(record.mappings) ? record.mappings : [];
    if (!record.mapped || mappings.length === 0) return null;

    const visibleMappings = mappings.slice(0, 3);
    const mappingText = mappings
      .map((mapping) => {
        const channelName = mapping.channel_name || `#${mapping.channel_id}`;
        return `${channelName}: ${mapping.source} -> ${mapping.target}`;
      })
      .join('\n');

    return (
      <Tooltip content={<pre className='whitespace-pre-wrap mb-0'>{mappingText}</pre>}>
        <div className='flex flex-wrap gap-1'>
          {visibleMappings.map((mapping) => (
            <Tag
              key={`${mapping.channel_id}-${mapping.source}-${mapping.target}`}
              size='small'
              color='blue'
            >
              {getMappingDisplayName(mapping, record.model_name)}
            </Tag>
          ))}
          {mappings.length > visibleMappings.length && (
            <Tag size='small' color='blue'>
              +{mappings.length - visibleMappings.length}
            </Tag>
          )}
        </div>
      </Tooltip>
    );
  };

  const renderChannelsColumn = (channels) => {
    const safeChannels = Array.isArray(channels) ? channels : [];
    if (safeChannels.length === 0) return <Text type='tertiary'>-</Text>;
    const visibleChannels = safeChannels.slice(0, 3);
    const channelText = safeChannels
      .map(
        (channel) =>
          `${getChannelName(channel)} ${getModelTestStatusText(channel.test_status, t)}`,
      )
      .join('\n');
    return (
      <Tooltip content={<pre className='whitespace-pre-wrap mb-0'>{channelText}</pre>}>
        <div className='flex flex-wrap gap-1'>
          {visibleChannels.map((channel) => (
            <Tag
              key={channel.id}
              size='small'
              color={getModelTestStatusColor(channel.test_status)}
            >
              {getChannelName(channel)}
            </Tag>
          ))}
          {safeChannels.length > visibleChannels.length && (
            <Tag size='small'>+{safeChannels.length - visibleChannels.length}</Tag>
          )}
        </div>
      </Tooltip>
    );
  };

  const renderPersistedTestStatus = (row) => {
    const statusText = getModelTestStatusText(row.testStatus, t);
    const lines = [statusText];
    if (row.testTime) {
      lines.push(`${t('测试时间')}：${new Date(row.testTime * 1000).toLocaleString()}`);
    }
    if (row.responseTime) {
      lines.push(`${t('请求时长: ${time}s').replace('${time}', (row.responseTime / 1000).toFixed(2))}`);
    }
    if (row.testError) {
      lines.push(`${t('错误信息')}：${row.testError}`);
    }
    if (row.testResponse) {
      lines.push(`${t('返回信息')}：${row.testResponse}`);
    }
    return (
      <Tooltip content={<pre className='whitespace-pre-wrap mb-0'>{lines.join('\n')}</pre>}>
        <Tag color={getModelTestStatusColor(row.testStatus)} shape='circle'>
          {statusText}
        </Tag>
      </Tooltip>
    );
  };

  const renderEditTable = (record) => {
    const editRows = getEditRows(record);
    if (editRows.length === 0) {
      return <Empty description={t('当前模型暂无可编辑渠道')} />;
    }

    return (
      <div className='models-edit-panel'>
        <Table
          size='small'
          pagination={false}
          rowKey='key'
          dataSource={editRows}
          columns={[
            {
              title: t('模型名称'),
              dataIndex: 'sourceModel',
              render: (sourceModel) => <Text strong>{sourceModel}</Text>,
            },
            {
              title: t('所属渠道'),
              dataIndex: 'channelName',
              render: (channelName, row) => `${channelName} (#${row.channelId})`,
            },
            {
              title: t('实际上游模型'),
              dataIndex: 'targetModel',
              render: (targetModel, row) => (
                <Input
                  placeholder={t('留空则不设置模型映射')}
                  value={mappingInputs[getMappingInputKey(row)] ?? targetModel}
                  onChange={(value) => setMappingValue(row, value)}
                />
              ),
            },
            {
              title: t('状态'),
              dataIndex: 'testStatus',
              width: 120,
              render: (_, row) => renderPersistedTestStatus(row),
            },
            {
              title: '',
              dataIndex: 'operate',
              width: 120,
              render: (_, row) => {
                const inputKey = getMappingInputKey(row);
                return (
                  <Button
                    size='small'
                    type='primary'
                    loading={savingMappingKeys.has(inputKey)}
                    onClick={() => saveMapping(row)}
                  >
                    {t('保存映射')}
                  </Button>
                );
              },
            },
          ]}
        />
      </div>
    );
  };

  const columns = [
    {
      title: t('模型名称'),
      dataIndex: 'model_name',
      width: 300,
      render: (modelName) => (
        <span className='cursor-pointer' onClick={() => copyModels(modelName)}>
          {renderModelTag(modelName, { shape: 'circle' })}
        </span>
      ),
    },
    {
      title: t('模型映射'),
      dataIndex: 'mapped',
      width: 260,
      render: (_, record) => renderMappingColumn(record),
    },
    {
      title: t('渠道'),
      dataIndex: 'channels',
      width: 260,
      render: renderChannelsColumn,
    },
    {
      title: '',
      dataIndex: 'operate',
      width: 220,
      fixed: 'right',
      render: (_, record) => {
        const testing = getTestItems(record).some((item) => testingItemKeys.has(item.key));
        return (
          <div className='flex items-center justify-end gap-2'>
            <SplitButtonGroup aria-label={t('测试选项')}>
              <Button
                type='tertiary'
                size='small'
                loading={testing}
                onClick={() => batchTestRecord(record)}
              >
                {t('测试')}
              </Button>
              <Button
                type='tertiary'
                size='small'
                icon={<IconTreeTriangleDown />}
                onClick={() => openTestOptions(record)}
                aria-label={t('测试选项')}
              />
            </SplitButtonGroup>
            <Button type='tertiary' size='small' onClick={() => openEditModel(record)}>
              {t('编辑')}
            </Button>
          </div>
        );
      },
    },
  ];

  const renderTestStatus = (item) => {
    if (testingItemKeys.has(item.key)) {
      return (
        <Tag color='blue' shape='circle'>
          {t('测试中')}
        </Tag>
      );
    }
    const result = modelTestResults[item.key];
    if (!result) {
      return (
        <Tag color='grey' shape='circle'>
          {t('未开始')}
        </Tag>
      );
    }
    return (
      <div className='flex flex-col gap-1'>
        <div className='flex items-center gap-2'>
          <Tag color={result.success ? 'green' : 'red'} shape='circle'>
            {result.success ? t('成功') : t('失败')}
          </Tag>
          {result.success && (
            <Text type='tertiary'>
              {t('请求时长: ${time}s').replace('${time}', result.time.toFixed(2))}
            </Text>
          )}
        </div>
        {!result.success && result.message && (
          <Text type='danger' size='small' className='break-all'>
            {result.message}
          </Text>
        )}
        {result.success && result.response && (
          <Text type='tertiary' size='small' className='break-all'>
            {t('返回信息')}：{result.response}
          </Text>
        )}
      </div>
    );
  };

  const renderHeader = () => (
    <Card
      className='!rounded-2xl shadow-sm border-0'
      cover={
        <div
          className='relative'
          style={{
            backgroundImage:
              "linear-gradient(0deg, rgba(37, 99, 235, 0.82), rgba(37, 99, 235, 0.82)), url('/cover-4.webp')",
            backgroundSize: 'cover',
            backgroundPosition: 'center',
          }}
        >
          <div className='flex items-center justify-between p-4'>
            <div className='min-w-0 flex-1 mr-4'>
              <div className='flex flex-wrap items-center gap-2 mb-2'>
                <h2 className='text-xl font-bold text-white mb-0'>{t('模型列表')}</h2>
                <Tag
                  shape='circle'
                  size='small'
                  style={{ backgroundColor: 'rgba(255,255,255,0.95)' }}
                >
                  {t('共 {{count}} 个模型', { count: filteredModels.length })}
                </Tag>
              </div>
              <Paragraph
                className='text-sm leading-relaxed !mb-0'
                style={{ color: 'rgba(255,255,255,0.9)' }}
                ellipsis={{ rows: 2 }}
              >
                {t('查看所有可用的AI模型供应商，包括众多知名供应商的模型。')}
              </Paragraph>
            </div>
            <div className='w-16 h-16 rounded-2xl bg-white/90 shadow-md flex items-center justify-center'>
              <Avatar size='large'>AI</Avatar>
            </div>
          </div>
        </div>
      }
    >
      <div className='flex flex-col md:flex-row items-stretch md:items-center gap-2'>
        <Input
          prefix={<IconSearch />}
          placeholder={t('模糊搜索模型名称')}
          value={searchValue}
          onChange={setSearchValue}
          showClear
        />
        <Button
          theme='outline'
          type='primary'
          icon={<IconCopy />}
          disabled={selectedRowKeys.length === 0}
          onClick={() => copyModels(selectedRowKeys)}
        >
          {t('复制')}
        </Button>
        <Dropdown
          trigger='click'
          position='bottomRight'
          render={
            <Dropdown.Menu>
              <Dropdown.Item onClick={batchTestSelectedModels} disabled={selectedRowKeys.length === 0}>
                {t('测试所选模型')}
              </Dropdown.Item>
            </Dropdown.Menu>
          }
        >
          <Button disabled={selectedRowKeys.length === 0} loading={isBatchTesting}>
            {t('批量操作')}
          </Button>
        </Dropdown>
        <Button icon={<IconRefresh />} onClick={fetchModels} loading={loading}>
          {t('刷新')}
        </Button>
      </div>
    </Card>
  );

  return (
    <>
      <Layout className='models-layout'>
        <Content className='models-content p-2 gap-2'>
          {renderHeader()}
          <Spin spinning={loading}>
            <Card className='models-table-card !rounded-2xl shadow-sm'>
              <Table
                columns={columns}
                dataSource={filteredModels}
                rowKey='model_name'
                pagination={false}
                scroll={{ x: 'max-content' }}
                rowSelection={{
                  selectedRowKeys,
                  onChange: (keys) =>
                    setSelectedRowKeys(Array.isArray(keys) ? keys : []),
                }}
                empty={<Empty description={t('暂无模型')} />}
              />
            </Card>
          </Spin>
        </Content>
      </Layout>

      <SideSheet
        placement='right'
        title={
          <div className='flex items-center justify-between w-full'>
            <div className='flex items-center gap-2 min-w-0'>
              <Tag color='blue' shape='circle'>
                {t('编辑')}
              </Tag>
              <Title heading={4} className='m-0 truncate'>
                {currentEditingModel?.model_name || t('模型详细信息')}
              </Title>
            </div>
          </div>
        }
        bodyStyle={{ padding: 0 }}
        visible={Boolean(currentEditingModel)}
        width={isMobile ? '100%' : 600}
        footer={
          <div className='flex justify-end items-center gap-2'>
            <Button
              theme='light'
              type='primary'
              icon={<IconClose />}
              onClick={() => setEditingModelRecord(null)}
            >
              {t('取消')}
            </Button>
          </div>
        }
        closeIcon={null}
        onCancel={() => setEditingModelRecord(null)}
      >
        {currentEditingModel && (
          <div className='models-edit-sidesheet p-4 flex flex-col gap-4'>
            <Card className='!rounded-xl'>
              <div className='flex flex-col gap-3'>
                <div>
                  <Text type='tertiary'>{t('模型名称')}</Text>
                  <div className='mt-1'>
                    {renderModelTag(currentEditingModel.model_name, { shape: 'circle' })}
                  </div>
                </div>
                <div>
                  <Text type='tertiary'>{t('模型映射')}</Text>
                  <div className='mt-1'>
                    {renderMappingColumn(currentEditingModel) || <Text type='tertiary'>-</Text>}
                  </div>
                </div>
                <div>
                  <Text type='tertiary'>{t('渠道')}</Text>
                  <div className='mt-1'>
                    {renderChannelsColumn(currentEditingModel.channels)}
                  </div>
                </div>
              </div>
            </Card>
            <Card className='!rounded-xl' title={t('模型详细信息')}>
              {renderEditTable(currentEditingModel)}
            </Card>
          </div>
        )}
      </SideSheet>

      <Modal
        title={
          testModalRecord ? (
            <div className='flex flex-col gap-2 w-full'>
              <div className='flex items-center gap-2'>
                <Text strong className='!text-[var(--semi-color-text-0)] !text-base'>
                  {testModalItems[0]?.displayModel || testModalRecord.model_name} {t('测试')}
                </Text>
                <Text type='tertiary' size='small'>
                  {t('共')} {testModalItems.length} {t('个模型')}
                </Text>
              </div>
            </div>
          ) : null
        }
        visible={Boolean(testModalRecord)}
        onCancel={closeTestModal}
        footer={
          <div className='flex justify-end gap-2'>
            <Button
              type={isBatchTesting ? 'danger' : 'tertiary'}
              onClick={closeTestModal}
            >
              {isBatchTesting ? t('停止测试') : t('取消')}
            </Button>
            <Button
              onClick={() => batchTestItems(selectedFilteredTestItems)}
              loading={isBatchTesting}
              disabled={isBatchTesting}
            >
              {isBatchTesting
                ? t('测试中...')
                : t('批量测试${count}个模型').replace(
                    '${count}',
                    selectedFilteredTestItems.length,
                  )}
            </Button>
          </div>
        }
        maskClosable={!isBatchTesting}
        size={isMobile ? 'full-width' : 'large'}
        className='!rounded-lg'
      >
        {testModalRecord && (
          <div className='model-test-scroll'>
            <div className='flex flex-col sm:flex-row sm:items-center gap-2 w-full mb-2'>
              <div className='flex items-center gap-2 flex-1 min-w-0'>
                <Text strong className='shrink-0'>
                  {t('端点类型')}:
                </Text>
                <Select
                  value={selectedEndpointType}
                  onChange={setSelectedEndpointType}
                  optionList={endpointTypeOptions}
                  className='!w-full min-w-0'
                  placeholder={t('选择端点类型')}
                />
              </div>
              <div className='flex items-center justify-between sm:justify-end gap-2 shrink-0'>
                <Text strong className='shrink-0'>
                  {t('流式')}:
                </Text>
                <Switch
                  checked={isStreamTest}
                  onChange={setIsStreamTest}
                  size='small'
                  disabled={streamToggleDisabled}
                  aria-label={t('流式')}
                />
              </div>
            </div>

            <Banner
              type='info'
              closeIcon={null}
              icon={<IconInfoCircle />}
              className='!rounded-lg mb-2'
              description={t(
                  '说明：可切换流式或非流式请求；若渠道不支持当前测试方式，可能出现测试失败，请以实际使用为准。',
              )}
            />

            <Input
              placeholder={t('搜索模型...')}
              value={modelSearchKeyword}
              onChange={(value) => {
                setModelSearchKeyword(value);
                setModelTablePage(1);
                setSelectedTestItemKeys([]);
              }}
              className='!w-full mb-2'
              prefix={<IconSearch />}
              showClear
            />

            <Table
              columns={[
                {
                  title: t('模型名称'),
                  dataIndex: 'sourceModel',
                  render: (sourceModel) => <Text strong>{sourceModel}</Text>,
                },
                {
                  title: t('所属渠道'),
                  dataIndex: 'channelName',
                  render: (channelName, item) => `${channelName} (#${item.channelId})`,
                },
                {
                  title: t('状态'),
                  dataIndex: 'status',
                  render: (_, item) => renderTestStatus(item),
                },
                {
                  title: t('测试'),
                  dataIndex: 'operate',
                  width: 100,
                  render: (_, item) => (
                    <Button
                      type='tertiary'
                      size='small'
                      loading={testingItemKeys.has(item.key)}
                      onClick={() => testItem(item)}
                    >
                      {t('测试')}
                    </Button>
                  ),
                },
              ]}
              dataSource={pagedTestModalItems}
              rowKey='key'
              rowSelection={{
                selectedRowKeys: selectedTestItemKeys,
                onChange: (keys) => {
                  if (allSelectingRef.current) {
                    allSelectingRef.current = false;
                    return;
                  }
                  setSelectedTestItemKeys(keys);
                },
                onSelectAll: (checked) => {
                  allSelectingRef.current = true;
                  const pagedKeys = pagedTestModalItems.map((item) => item.key);
                  if (checked) {
                    setSelectedTestItemKeys((prev) =>
                      Array.from(new Set([...prev, ...pagedKeys])),
                    );
                  } else {
                    setSelectedTestItemKeys((prev) =>
                      prev.filter((key) => !pagedKeys.includes(key)),
                    );
                  }
                },
              }}
              pagination={{
                currentPage: modelTablePage,
                pageSize: MODEL_TABLE_PAGE_SIZE,
                total: filteredTestModalItems.length,
                showSizeChanger: false,
                onPageChange: (page) => setModelTablePage(page),
              }}
            />
          </div>
        )}
      </Modal>
    </>
  );
};

export default Models;
