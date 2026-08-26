import React, { useEffect, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Checkbox,
  Divider,
  Modal,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { Download, RefreshCw, ShieldAlert, Upload } from 'lucide-react';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';

const BACKUP_FORMAT = 'new-api-lite.config-backup';
const BACKUP_VERSION = 1;

const ConfigBackupSetting = () => {
  const { t } = useTranslation();
  const fileInputRef = useRef(null);
  const [categories, setCategories] = useState([]);
  const [selectedExportCategories, setSelectedExportCategories] = useState([]);
  const [selectedRestoreCategories, setSelectedRestoreCategories] = useState([]);
  const [importedBackup, setImportedBackup] = useState(null);
  const [importedFileName, setImportedFileName] = useState('');
  const [loading, setLoading] = useState(false);

  const loadCategories = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/option/config_backup/categories');
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      const nextCategories = res.data.data || [];
      setCategories(nextCategories);
      setSelectedExportCategories(
        nextCategories.filter((item) => !item.sensitive).map((item) => item.key),
      );
    } catch (error) {
      showError(t('加载失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadCategories();
  }, []);

  const hasCategory = (selected, categoryKey) => selected.includes(categoryKey);
  const includesSensitive = (selected) =>
    categories.some((item) => item.sensitive && hasCategory(selected, item.key));
  const includesChannels = (selected) => hasCategory(selected, 'channels');

  const downloadBackup = async (includeSensitive) => {
    setLoading(true);
    try {
      const response = await API.post(
        '/api/option/config_backup/export',
        {
          categories: selectedExportCategories,
          include_sensitive: includeSensitive,
        },
        { responseType: 'blob' },
      );
      const blob = new Blob([response.data], { type: 'application/json' });
      const contentDisposition = response.headers?.['content-disposition'] || '';
      const matchedName = contentDisposition.match(/filename=([^;]+)/i);
      const filename = matchedName
        ? matchedName[1].replace(/"/g, '')
        : `new-api-lite-config-backup-${new Date().toISOString().replace(/[:.]/g, '-')}.json`;
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);
      showSuccess(t('配置备份已下载'));
    } catch (error) {
      showError(t('导出配置失败'));
    } finally {
      setLoading(false);
    }
  };

  const handleExport = () => {
    if (selectedExportCategories.length === 0) {
      showError(t('请至少选择一个配置类别'));
      return;
    }
    const sensitive = includesSensitive(selectedExportCategories);
    if (!sensitive) {
      downloadBackup(false);
      return;
    }
    Modal.confirm({
      title: t('导出敏感配置'),
      content: t(
        '所选内容包含渠道密钥或系统凭据。导出的 JSON 将包含明文敏感信息，请仅保存到受信任且受保护的位置。是否继续？',
      ),
      okText: t('确认导出'),
      cancelText: t('取消'),
      okButtonProps: { type: 'danger' },
      onOk: () => downloadBackup(true),
    });
  };

  const handleImportClick = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = (event) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) {
      return;
    }
    if (file.size > 16 * 1024 * 1024) {
      showError(t('备份文件不能超过 16 MB'));
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      try {
        const backup = JSON.parse(reader.result);
        if (
          backup?.format !== BACKUP_FORMAT ||
          backup?.version !== BACKUP_VERSION ||
          !Array.isArray(backup?.categories)
        ) {
          showError(t('不是受支持的配置备份文件'));
          return;
        }
        setImportedBackup(backup);
        setImportedFileName(file.name);
        setSelectedRestoreCategories(backup.categories);
        showSuccess(t('备份文件已载入，请选择需要还原的类别'));
      } catch (error) {
        showError(t('无法解析配置备份文件'));
      }
    };
    reader.onerror = () => showError(t('读取配置备份文件失败'));
    reader.readAsText(file);
  };

  const performRestore = async () => {
    setLoading(true);
    try {
      const res = await API.post('/api/option/config_backup/restore', {
        backup: importedBackup,
        categories: selectedRestoreCategories,
        confirm_sensitive: includesSensitive(selectedRestoreCategories),
        confirm_channels: includesChannels(selectedRestoreCategories),
      });
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      const data = res.data.data || {};
      showSuccess(
        t('配置还原成功') +
          `：${t('设置')} ${data.options_restored || 0}，${t('渠道')} ${data.channels_restored || 0}`,
      );
      setImportedBackup(null);
      setImportedFileName('');
      setSelectedRestoreCategories([]);
      await loadCategories();
    } catch (error) {
      showError(t('还原配置失败'));
    } finally {
      setLoading(false);
    }
  };

  const handleRestore = () => {
    if (!importedBackup) {
      showError(t('请先导入配置备份文件'));
      return;
    }
    if (selectedRestoreCategories.length === 0) {
      showError(t('请至少选择一个配置类别'));
      return;
    }
    const includesSensitiveData = includesSensitive(selectedRestoreCategories);
    const replacesChannels = includesChannels(selectedRestoreCategories);
    const notices = [t('仅会还原当前勾选的类别，未勾选的当前配置保持不变。')];
    if (includesSensitiveData) {
      notices.push(t('将覆盖当前 SMTP、Turnstile、Worker 凭据或渠道密钥。'));
    }
    if (replacesChannels) {
      notices.push(t('将替换当前全部渠道和模型能力配置，建议先下载当前备份。'));
    }
    Modal.confirm({
      title: t('确认还原配置'),
      content: notices.join('\n'),
      okText: t('确认还原'),
      cancelText: t('取消'),
      okButtonProps: { type: 'danger' },
      onOk: performRestore,
    });
  };

  const renderCategories = (selected, setSelected, availableKeys) => (
    <Checkbox.Group value={selected} onChange={setSelected}>
      <div className='grid grid-cols-1 gap-3 md:grid-cols-2'>
        {categories
          .filter((item) => !availableKeys || availableKeys.includes(item.key))
          .map((item) => (
            <Card key={item.key} className='!border-gray-200'>
              <Checkbox value={item.key}>
                <div className='flex flex-wrap items-center gap-2'>
                  <Typography.Text strong>{t(item.name)}</Typography.Text>
                  {item.sensitive && <Tag color='red'>{t('敏感信息')}</Tag>}
                  {item.destructive && <Tag color='orange'>{t('覆盖现有渠道')}</Tag>}
                  <Tag color='blue'>{item.item_count}</Tag>
                </div>
                <Typography.Paragraph className='!mb-0 !mt-2 !text-xs !text-gray-500'>
                  {t(item.description)}
                </Typography.Paragraph>
              </Checkbox>
            </Card>
          ))}
      </div>
    </Checkbox.Group>
  );

  return (
    <Spin spinning={loading} size='large'>
      <Card title={t('配置备份与还原')} style={{ marginTop: '10px' }}>
        <Banner
          type='info'
          closeIcon={null}
          description={t(
            '备份仅包含当前有效的系统配置与可选渠道配置；不会包含用户账户、API Token、2FA、请求日志、tokens 使用统计、性能指标或 SQLite 数据库文件。',
          )}
        />
        <div className='mt-5'>
          <Typography.Title heading={5}>{t('导出配置备份')}</Typography.Title>
          <Typography.Paragraph className='!text-gray-500'>
            {t('选择需要保存的配置类别。敏感凭据和渠道密钥默认不选择，并会在下载前要求确认。')}
          </Typography.Paragraph>
          {renderCategories(selectedExportCategories, setSelectedExportCategories)}
          <div className='mt-4 flex flex-wrap gap-3'>
            <Button icon={<Download size={16} />} type='primary' onClick={handleExport}>
              {t('下载配置备份')}
            </Button>
            <Button icon={<RefreshCw size={16} />} theme='borderless' onClick={loadCategories}>
              {t('刷新类别')}
            </Button>
          </div>
        </div>

        <Divider margin='24px' />

        <div>
          <Typography.Title heading={5}>{t('导入并还原配置')}</Typography.Title>
          <Typography.Paragraph className='!text-gray-500'>
            {t('选择本系统导出的 JSON 备份文件。载入后可再次选择本次需要还原的类别。')}
          </Typography.Paragraph>
          <input
            ref={fileInputRef}
            type='file'
            accept='application/json,.json'
            onChange={handleFileChange}
            style={{ display: 'none' }}
          />
          <Button icon={<Upload size={16} />} onClick={handleImportClick}>
            {t('选择备份文件')}
          </Button>
          {importedBackup && (
            <div className='mt-4'>
              <Banner
                type='warning'
                closeIcon={null}
                icon={<ShieldAlert size={18} />}
                description={`${t('已载入')} ${importedFileName} · ${t('创建时间')} ${new Date(importedBackup.created_at).toLocaleString()}`}
              />
              <div className='mt-4'>
                {renderCategories(
                  selectedRestoreCategories,
                  setSelectedRestoreCategories,
                  importedBackup.categories,
                )}
              </div>
              <Button className='mt-4' type='danger' onClick={handleRestore}>
                {t('还原所选配置')}
              </Button>
            </div>
          )}
        </div>
      </Card>
    </Spin>
  );
};

export default ConfigBackupSetting;
