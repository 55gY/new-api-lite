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

import React from 'react';
import {
  Modal,
  Button,
  Input,
  Table,
  Tag,
  Typography,
  Select,
  Switch,
  Banner,
  Tooltip,
} from '@douyinfe/semi-ui';
import { IconSearch, IconInfoCircle } from '@douyinfe/semi-icons';
import { Settings } from 'lucide-react';
import { API, copy, showError, showInfo, showSuccess } from '../../../../helpers';
import { MODEL_TABLE_PAGE_SIZE } from '../../../../constants';

const MODEL_TEST_STATUS = {
  untested: 0,
  available: 1,
  unavailable: 2,
};

const MODEL_STATUS = {
  enabled: 1,
  disabled: 2,
  autoDisabled: 3,
};

const getModelStatusText = (status, t) => {
  if (status === MODEL_STATUS.disabled) return t('已禁用');
  if (status === MODEL_STATUS.autoDisabled) return t('自动禁用');
  return t('已启用');
};

const getModelStatusColor = (status) => {
  if (status === MODEL_STATUS.disabled) return 'red';
  if (status === MODEL_STATUS.autoDisabled) return 'orange';
  return 'green';
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

const ModelTestModal = ({
  showModelTestModal,
  currentTestChannel,
  handleCloseModal,
  isBatchTesting,
  batchTestModels,
  modelSearchKeyword,
  setModelSearchKeyword,
  selectedModelKeys,
  setSelectedModelKeys,
  modelTestResults,
  setModelTestResults,
  testingModels,
  testChannel,
  modelTablePage,
  setModelTablePage,
  selectedEndpointType,
  setSelectedEndpointType,
  isStreamTest,
  setIsStreamTest,
  allSelectingRef,
  isMobile,
  refresh,
  setCurrentTestChannel,
  t,
}) => {
  const hasChannel = Boolean(currentTestChannel);
  const [abilities, setAbilities] = React.useState([]);
  const [abilitiesLoading, setAbilitiesLoading] = React.useState(false);
  const [updatingStatusKeys, setUpdatingStatusKeys] = React.useState(new Set());
  const streamToggleDisabled = [
    'embeddings',
    'image-generation',
    'jina-rerank',
    'openai-response-compact',
  ].includes(selectedEndpointType);

  React.useEffect(() => {
    if (streamToggleDisabled && isStreamTest) {
      setIsStreamTest(false);
    }
  }, [streamToggleDisabled, isStreamTest, setIsStreamTest]);

  const fetchAbilities = React.useCallback(async () => {
    if (!showModelTestModal || !currentTestChannel?.id) {
      setAbilities([]);
      return;
    }
    setAbilitiesLoading(true);
    try {
      const res = await API.get(`/api/channel/${currentTestChannel.id}/abilities`);
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setAbilities(Array.isArray(data) ? data : []);
    } catch (error) {
      showError(error.message || t('获取模型状态失败'));
    } finally {
      setAbilitiesLoading(false);
    }
  }, [currentTestChannel?.id, showModelTestModal, t]);

  React.useEffect(() => {
    fetchAbilities();
  }, [fetchAbilities]);

  const abilityByModel = React.useMemo(() => {
    const map = new Map();
    abilities.forEach((ability) => {
      if (ability?.model) {
        map.set(ability.model, ability);
      }
    });
    return map;
  }, [abilities]);

  const channelModels = hasChannel && currentTestChannel.models
    ? currentTestChannel.models
        .split(',')
        .map((model) => model.trim())
        .filter(Boolean)
    : [];

  const filteredModels = channelModels
        .filter((model) =>
          model.toLowerCase().includes(modelSearchKeyword.toLowerCase()),
        );

  const renderModelStatus = (status) => (
    <Tag color={getModelStatusColor(status)} shape='circle'>
      {getModelStatusText(status, t)}
    </Tag>
  );

  const renderPersistedTestStatus = (record) => {
    const statusText = getModelTestStatusText(record.testStatus, t);
    const lines = [statusText];
    if (record.testTime) {
      lines.push(`${t('测试时间')}：${new Date(record.testTime * 1000).toLocaleString()}`);
    }
    if (record.responseTime) {
      lines.push(
        `${t('请求时长: ${time}s').replace('${time}', (record.responseTime / 1000).toFixed(2))}`,
      );
    }
    if (record.testError) {
      lines.push(`${t('错误信息')}：${record.testError}`);
    }
    if (record.testResponse) {
      lines.push(`${t('返回信息')}：${record.testResponse}`);
    }
    return (
      <Tooltip content={<pre className='whitespace-pre-wrap mb-0'>{lines.join('\n')}</pre>}>
        <Tag color={getModelTestStatusColor(record.testStatus)} shape='circle'>
          {statusText}
        </Tag>
      </Tooltip>
    );
  };

  const updateModelStatus = async (record, status) => {
    const statusKey = `${currentTestChannel.id}-${record.model}`;
    setUpdatingStatusKeys((prev) => new Set([...prev, statusKey]));
    try {
      const res = await API.put(`/api/channel/${currentTestChannel.id}/abilities`, {
        model: record.model,
        status,
      });
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(t('操作成功'));
      await fetchAbilities();
    } catch (error) {
      showError(error.message || t('操作失败'));
    } finally {
      setUpdatingStatusKeys((prev) => {
        const next = new Set(prev);
        next.delete(statusKey);
        return next;
      });
    }
  };

  const runModelTest = async (record) => {
    await testChannel(
      currentTestChannel,
      record.model,
      selectedEndpointType,
      isStreamTest,
    );
    await fetchAbilities();
  };

  const runBatchTestModels = async () => {
    await batchTestModels();
    await fetchAbilities();
  };

  const deleteModel = (record) => {
    Modal.confirm({
      title: t('删除模型'),
      content: t('确定要从当前渠道中删除模型 ${model} 吗？').replace(
        '${model}',
        record.model,
      ),
      onOk: async () => {
        try {
          const res = await API.delete(`/api/channel/${currentTestChannel.id}/models`, {
            data: { model: record.model },
          });
          const { success, message, data } = res.data;
          if (!success) {
            showError(message);
            return;
          }
          showSuccess(t('删除成功'));
          setCurrentTestChannel?.(data || {
            ...currentTestChannel,
            models: channelModels.filter((model) => model !== record.model).join(','),
          });
          setSelectedModelKeys((prev) => prev.filter((model) => model !== record.model));
          setModelTestResults?.((prev) => {
            const next = { ...prev };
            delete next[`${currentTestChannel.id}-${record.model}`];
            return next;
          });
          await refresh?.();
          await fetchAbilities();
        } catch (error) {
          showError(error.message || t('删除失败'));
        }
      },
    });
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
    {
      value: 'gemini',
      label: 'Gemini (/v1beta/models/{model}:generateContent)',
    },
    { value: 'jina-rerank', label: 'Jina Rerank (/v1/rerank)' },
    {
      value: 'image-generation',
      label: t('图像生成') + ' (/v1/images/generations)',
    },
    { value: 'embeddings', label: 'Embeddings (/v1/embeddings)' },
  ];

  const handleCopySelected = () => {
    if (selectedModelKeys.length === 0) {
      showError(t('请先选择模型！'));
      return;
    }
    copy(selectedModelKeys.join(',')).then((ok) => {
      if (ok) {
        showSuccess(
          t('已复制 ${count} 个模型').replace(
            '${count}',
            selectedModelKeys.length,
          ),
        );
      } else {
        showError(t('复制失败，请手动复制'));
      }
    });
  };

  const handleSelectSuccess = () => {
    if (!currentTestChannel) return;
    const successKeys = channelModels
      .filter((m) => m.toLowerCase().includes(modelSearchKeyword.toLowerCase()))
      .filter((m) => {
        const result = modelTestResults[`${currentTestChannel.id}-${m}`];
        return result && result.success;
      });
    if (successKeys.length === 0) {
      showInfo(t('暂无成功模型'));
    }
    setSelectedModelKeys(successKeys);
  };

  const columns = [
    {
      title: t('模型名称'),
      dataIndex: 'model',
      render: (text) => (
        <div className='flex items-center'>
          <Typography.Text strong>{text}</Typography.Text>
        </div>
      ),
    },
    {
      title: t('模型状态'),
      dataIndex: 'modelStatus',
      width: 120,
      render: renderModelStatus,
    },
    {
      title: t('测试状态'),
      dataIndex: 'testStatus',
      render: (text, record) => {
        const testResult =
          modelTestResults[`${currentTestChannel.id}-${record.model}`];
        const isTesting = testingModels.has(record.model);

        if (isTesting) {
          return (
            <Tag color='blue' shape='circle'>
              {t('测试中')}
            </Tag>
          );
        }

        if (!testResult) {
          return renderPersistedTestStatus(record);
        }

        return (
          <div className='flex flex-col gap-1'>
            <div className='flex items-center gap-2'>
              <Tag color={testResult.success ? 'green' : 'red'} shape='circle'>
                {testResult.success ? t('成功') : t('失败')}
              </Tag>
              {testResult.success && (
                <Typography.Text type='tertiary'>
                  {t('请求时长: ${time}s').replace(
                    '${time}',
                    testResult.time.toFixed(2),
                  )}
                </Typography.Text>
              )}
            </div>
            {testResult.success && testResult.response && (
              <Typography.Text
                type='tertiary'
                size='small'
                className='break-all'
                style={{ maxWidth: '400px', fontSize: '12px' }}
              >
                {t('返回信息')}：{testResult.response}
              </Typography.Text>
            )}
            {!testResult.success && testResult.message && (
              <div className='flex flex-col gap-1'>
                <Typography.Text
                  type='danger'
                  size='small'
                  className='break-all'
                  style={{ maxWidth: '400px', fontSize: '12px' }}
                >
                  {testResult.message}
                </Typography.Text>
                {testResult.errorCode === 'model_price_error' && (
                  <Button
                    size='small'
                    theme='light'
                    type='warning'
                    icon={<Settings size={12} />}
                    onClick={() => window.open('/console/setting?tab=ratio', '_blank')}
                    style={{ width: 'fit-content' }}
                  >
                    {t('前往设置')}
                  </Button>
                )}
              </div>
            )}
          </div>
        );
      },
    },
    {
      title: '',
      dataIndex: 'operate',
      render: (text, record) => {
        const isTesting = testingModels.has(record.model);
        const statusKey = `${currentTestChannel.id}-${record.model}`;
        const nextStatus =
          record.modelStatus !== MODEL_STATUS.enabled
            ? MODEL_STATUS.enabled
            : MODEL_STATUS.disabled;
        return (
          <div className='flex items-center gap-2'>
            <Button
              type='tertiary'
              onClick={() => updateModelStatus(record, nextStatus)}
              loading={updatingStatusKeys.has(statusKey)}
              size='small'
            >
              {nextStatus === MODEL_STATUS.enabled ? t('启用') : t('禁用')}
            </Button>
            <Button
              type='tertiary'
              onClick={() => runModelTest(record)}
              loading={isTesting}
              size='small'
            >
              {t('测试')}
            </Button>
            <Button
              type='danger'
              theme='light'
              onClick={() => deleteModel(record)}
              disabled={isTesting || isBatchTesting}
              size='small'
            >
              {t('删除')}
            </Button>
          </div>
        );
      },
    },
  ];

  const dataSource = (() => {
    if (!hasChannel) return [];
    const start = (modelTablePage - 1) * MODEL_TABLE_PAGE_SIZE;
    const end = start + MODEL_TABLE_PAGE_SIZE;
    return filteredModels.slice(start, end).map((model) => {
      const ability = abilityByModel.get(model) || {};
      return {
        model,
        key: model,
        modelStatus: ability.status ?? MODEL_STATUS.enabled,
        testStatus: ability.test_status ?? MODEL_TEST_STATUS.untested,
        testTime: ability.test_time ?? 0,
        responseTime: ability.response_time ?? 0,
        testError: ability.test_error || '',
        testResponse: ability.test_response || '',
      };
    });
  })();

  return (
    <Modal
      title={
        hasChannel ? (
          <div className='flex flex-col gap-2 w-full'>
            <div className='flex items-center gap-2'>
              <Typography.Text
                strong
                className='!text-[var(--semi-color-text-0)] !text-base'
              >
                {currentTestChannel.name} {t('渠道的模型测试')}
              </Typography.Text>
              <Typography.Text type='tertiary' size='small'>
                {t('共')} {channelModels.length}{' '}
                {t('个模型')}
              </Typography.Text>
            </div>
          </div>
        ) : null
      }
      visible={showModelTestModal}
      onCancel={handleCloseModal}
      footer={
        hasChannel ? (
          <div className='flex justify-end'>
            {isBatchTesting ? (
              <Button type='danger' onClick={handleCloseModal}>
                {t('停止测试')}
              </Button>
            ) : (
              <Button type='tertiary' onClick={handleCloseModal}>
                {t('取消')}
              </Button>
            )}
            <Button
              onClick={runBatchTestModels}
              loading={isBatchTesting}
              disabled={isBatchTesting || abilitiesLoading}
            >
              {isBatchTesting
                ? t('测试中...')
                : t('批量测试${count}个模型').replace(
                    '${count}',
                    filteredModels.length,
                  )}
            </Button>
          </div>
        ) : null
      }
      maskClosable={!isBatchTesting}
      className='!rounded-lg'
      size={isMobile ? 'full-width' : 'large'}
    >
      {hasChannel && (
        <div className='model-test-scroll'>
          {/* Endpoint toolbar */}
          <div className='flex flex-col sm:flex-row sm:items-center gap-2 w-full mb-2'>
            <div className='flex items-center gap-2 flex-1 min-w-0'>
              <Typography.Text strong className='shrink-0'>
                {t('端点类型')}:
              </Typography.Text>
              <Select
                value={selectedEndpointType}
                onChange={setSelectedEndpointType}
                optionList={endpointTypeOptions}
                className='!w-full min-w-0'
                placeholder={t('选择端点类型')}
              />
            </div>
            <div className='flex items-center justify-between sm:justify-end gap-2 shrink-0'>
              <Typography.Text strong className='shrink-0'>
                {t('流式')}:
              </Typography.Text>
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
              '说明：本页可切换流式测试；若渠道仅支持特定返回方式，可能出现测试失败，请以实际使用为准。',
            )}
          />

          {/* 搜索与操作按钮 */}
          <div className='flex flex-col sm:flex-row sm:items-center gap-2 w-full mb-2'>
            <Input
              placeholder={t('搜索模型...')}
              value={modelSearchKeyword}
              onChange={(v) => {
                setModelSearchKeyword(v);
                setModelTablePage(1);
              }}
              className='!w-full sm:!flex-1'
              prefix={<IconSearch />}
              showClear
            />

            <div className='flex items-center justify-end gap-2'>
              <Button onClick={handleCopySelected}>{t('复制已选')}</Button>
              <Button type='tertiary' onClick={handleSelectSuccess}>
                {t('选择成功')}
              </Button>
            </div>
          </div>

          <Table
            columns={columns}
            dataSource={dataSource}
            loading={abilitiesLoading}
            rowSelection={{
              selectedRowKeys: selectedModelKeys,
              onChange: (keys) => {
                if (allSelectingRef.current) {
                  allSelectingRef.current = false;
                  return;
                }
                setSelectedModelKeys(keys);
              },
              onSelectAll: (checked) => {
                allSelectingRef.current = true;
                setSelectedModelKeys(checked ? filteredModels : []);
              },
            }}
            pagination={{
              currentPage: modelTablePage,
              pageSize: MODEL_TABLE_PAGE_SIZE,
              total: filteredModels.length,
              showSizeChanger: false,
              onPageChange: (page) => setModelTablePage(page),
            }}
          />
        </div>
      )}
    </Modal>
  );
};

export default ModelTestModal;
