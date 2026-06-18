/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation; either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useState, useRef, useMemo, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Tag, Input, Typography, Space } from '@douyinfe/semi-ui';
import { copy, showError, showSuccess } from '../../../../helpers';
import { selectFilter } from '../../../../helpers';

const { Text } = Typography;

/**
 * 模型标签输入组件
 *
 * 支持：
 * - 文本输入逗号分隔的模型名，回车/失焦时解析添加
 * - 已选模型以可移除标签展示
 * - 点击标签可复制模型名
 * - 下拉建议选择（可选）
 */
const ModelTagInput = ({
  value = [],
  onChange,
  modelOptions = [],
  placeholder,
  required = false,
  onSearch,
  actionButtons,
}) => {
  const { t } = useTranslation();
  const [inputValue, setInputValue] = useState('');
  const [showDropdown, setShowDropdown] = useState(false);
  const [highlightIndex, setHighlightIndex] = useState(-1);
  const inputRef = useRef(null);
  const dropdownRef = useRef(null);

  // 过滤下拉建议：排除已选模型，按输入关键词过滤
  const filteredSuggestions = useMemo(() => {
    const keyword = inputValue.trim();
    const selectedSet = new Set(value.map((m) => m.toLowerCase()));
    return modelOptions
      .filter((opt) => !selectedSet.has((opt.value || '').toLowerCase()))
      .filter((opt) => (keyword ? selectFilter(keyword, opt) : true))
      .slice(0, 50); // 限制显示数量
  }, [inputValue, modelOptions, value]);

  const addModels = useCallback(
    (modelsToAdd) => {
      const unique = [];
      const existingSet = new Set(value.map((m) => m.toLowerCase()));
      for (const m of modelsToAdd) {
        const trimmed = (m || '').trim();
        if (trimmed && !existingSet.has(trimmed.toLowerCase())) {
          unique.push(trimmed);
          existingSet.add(trimmed.toLowerCase());
        }
      }
      if (unique.length > 0) {
        onChange([...value, ...unique]);
      }
    },
    [value, onChange],
  );

  const removeModel = useCallback(
    (modelToRemove) => {
      onChange(value.filter((m) => m !== modelToRemove));
    },
    [value, onChange],
  );

  const handleInputConfirm = useCallback(() => {
    const raw = inputValue.trim();
    if (!raw) return;
    const models = raw.split(',').map((m) => m.trim()).filter(Boolean);
    addModels(models);
    setInputValue('');
    setShowDropdown(false);
    setHighlightIndex(-1);
  }, [inputValue, addModels]);

  const handleKeyDown = useCallback(
    (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        if (highlightIndex >= 0 && highlightIndex < filteredSuggestions.length) {
          addModels([filteredSuggestions[highlightIndex].value]);
          setInputValue('');
          setShowDropdown(false);
          setHighlightIndex(-1);
        } else {
          handleInputConfirm();
        }
      } else if (e.key === 'Backspace' && !inputValue && value.length > 0) {
        // 输入为空时按 Backspace 删除最后一个标签
        removeModel(value[value.length - 1]);
      } else if (e.key === 'ArrowDown') {
        e.preventDefault();
        setHighlightIndex((prev) =>
          prev < filteredSuggestions.length - 1 ? prev + 1 : 0,
        );
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setHighlightIndex((prev) =>
          prev > 0 ? prev - 1 : filteredSuggestions.length - 1,
        );
      } else if (e.key === 'Escape') {
        setShowDropdown(false);
        setHighlightIndex(-1);
      }
    },
    [
      inputValue,
      value,
      filteredSuggestions,
      highlightIndex,
      addModels,
      removeModel,
      handleInputConfirm,
    ],
  );

  const handleInputChange = useCallback(
    (val) => {
      setInputValue(val);
      if (onSearch) onSearch(val);
      setShowDropdown(true);
      setHighlightIndex(-1);
    },
    [onSearch],
  );

  const handleInputBlur = useCallback(() => {
    // 延迟关闭下拉，让点击建议项的事件先触发
    setTimeout(() => {
      setShowDropdown(false);
      setHighlightIndex(-1);
    }, 200);
  }, []);

  const handleInputFocus = useCallback(() => {
    if (inputValue.trim()) {
      setShowDropdown(true);
    }
  }, [inputValue]);

  const handleTagClick = useCallback(
    async (modelName, e) => {
      e.stopPropagation();
      const ok = await copy(modelName);
      if (ok) {
        showSuccess(t('已复制：{{name}}', { name: modelName }));
      } else {
        showError(t('复制失败'));
      }
    },
    [t],
  );

  const handleSuggestionMouseDown = useCallback(
    (modelValue, e) => {
      e.preventDefault(); // 防止触发 blur
      addModels([modelValue]);
      setInputValue('');
      setShowDropdown(false);
      setHighlightIndex(-1);
    },
    [addModels],
  );

  return (
    <div style={{ width: '100%' }}>
      {/* 标签展示区域 */}
      {value.length > 0 && (
        <div
          style={{
            marginBottom: 8,
            display: 'flex',
            flexWrap: 'wrap',
            gap: 4,
            maxHeight: 200,
            overflowY: 'auto',
          }}
        >
          {value.map((modelName) => (
            <Tag
              key={modelName}
              closable
              onClose={() => removeModel(modelName)}
              onClick={(e) => handleTagClick(modelName, e)}
              style={{ cursor: 'pointer', maxWidth: 300 }}
              title={modelName}
            >
              <span
                style={{
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  maxWidth: 260,
                  display: 'inline-block',
                }}
              >
                {modelName}
              </span>
            </Tag>
          ))}
        </div>
      )}

      {/* 输入框 + 下拉建议 */}
      <div style={{ position: 'relative' }}>
        <div style={{ display: 'flex', gap: 8, alignItems: 'flex-start' }}>
          <div style={{ flex: 1, position: 'relative' }}>
            <Input
              ref={inputRef}
              value={inputValue}
              onChange={handleInputChange}
              onKeyDown={handleKeyDown}
              onBlur={handleInputBlur}
              onFocus={handleInputFocus}
              placeholder={placeholder || t('输入模型名称，多个用逗号分隔，回车确认')}
              style={{ width: '100%' }}
              showClear
              onClear={() => {
                setInputValue('');
                setShowDropdown(false);
              }}
            />
            {/* 下拉建议 */}
            {showDropdown && filteredSuggestions.length > 0 && (
              <div
                ref={dropdownRef}
                style={{
                  position: 'absolute',
                  top: '100%',
                  left: 0,
                  right: 0,
                  zIndex: 1050,
                  backgroundColor: 'var(--semi-color-bg-0)',
                  border: '1px solid var(--semi-color-border)',
                  borderRadius: 4,
                  boxShadow: '0 2px 8px rgba(0, 0, 0, 0.15)',
                  maxHeight: 240,
                  overflowY: 'auto',
                  marginTop: 4,
                }}
              >
                {filteredSuggestions.map((opt, index) => (
                  <div
                    key={opt.value}
                    onMouseDown={(e) =>
                      handleSuggestionMouseDown(opt.value, e)
                    }
                    style={{
                      padding: '6px 12px',
                      cursor: 'pointer',
                      backgroundColor:
                        index === highlightIndex
                          ? 'var(--semi-color-fill-1)'
                          : 'transparent',
                      fontSize: 13,
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                    }}
                    onMouseEnter={() => setHighlightIndex(index)}
                  >
                    {opt.label || opt.value}
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* 操作按钮区域 */}
          {actionButtons && (
            <Space>
              {actionButtons}
            </Space>
          )}
        </div>

        {/* 底部提示 */}
        {value.length > 0 && (
          <Text
            className='block text-xs !text-semi-color-text-2'
            style={{ marginTop: 4 }}
          >
            {t('已选择 {{count}} 个模型', { count: value.length })}
            {' · '}
            {t('点击标签可复制，输入逗号分隔批量添加')}
          </Text>
        )}

        {/* 必填提示 */}
        {required && value.length === 0 && (
          <Text
            className='block text-xs'
            style={{ marginTop: 4, color: 'var(--semi-color-danger)' }}
          >
            {t('请选择模型')}
          </Text>
        )}
      </div>
    </div>
  );
};

export default ModelTagInput;
