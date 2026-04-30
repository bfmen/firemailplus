'use client';

import { useEffect, useState, useRef } from 'react';
import { FilterGroup, DateRangeFilter } from './filter-group';
import { useMailboxStore } from '@/lib/store';
import { Button } from '@/components/ui/button';
import { X } from 'lucide-react';
import type { SearchParams } from '@/hooks/use-search-emails';
import { apiClient } from '@/lib/api';
import type { Folder } from '@/types/email';

interface SearchFiltersProps {
  onFiltersChange: (filters: Partial<SearchParams>) => void;
}

export function SearchFilters({ onFiltersChange }: SearchFiltersProps) {
  const { accounts, folders, setAccounts, setFolders } = useMailboxStore();
  const isInitialMount = useRef(true);
  const onFiltersChangeRef = useRef(onFiltersChange);

  // 保持对最新回调函数的引用
  onFiltersChangeRef.current = onFiltersChange;

  // 本地筛选状态
  const [localFilters, setLocalFilters] = useState({
    account_ids: [] as number[],
    folder_ids: [] as number[],
    is_read: undefined as boolean | undefined,
    is_starred: undefined as boolean | undefined,
    is_important: undefined as boolean | undefined,
    has_attachment: undefined as boolean | undefined,
    date_range: { start: undefined, end: undefined } as { start?: string; end?: string },
  });

  // 刷新搜索页时补拉账户与文件夹，避免筛选项为空
  useEffect(() => {
    let cancelled = false;

    const loadAccountsIfNeeded = async () => {
      if (accounts.length > 0) return;
      try {
        const response = await apiClient.getEmailAccounts();
        if (!cancelled && response.success && response.data) {
          setAccounts(response.data);
        }
      } catch (error) {
        console.error('Failed to load accounts for search filters:', error);
      }
    };

    void loadAccountsIfNeeded();

    return () => {
      cancelled = true;
    };
  }, [accounts.length, setAccounts]);

  useEffect(() => {
    let cancelled = false;

    const mergeFolders = (responses: Array<{ success: boolean; data?: Folder[] }>) => {
      const folderMap = new Map<number, Folder>();
      responses.forEach((response) => {
        if (!response.success || !response.data) return;
        response.data.forEach((folder) => {
          folderMap.set(folder.id, folder);
        });
      });
      return Array.from(folderMap.values());
    };

    const loadFoldersIfNeeded = async () => {
      if (folders.length > 0) return;

      if (accounts.length === 0) return;

      try {
        const responses = await Promise.all(
          accounts.map((account) => apiClient.getFolders(account.id))
        );
        const merged = mergeFolders(responses);
        if (!cancelled && merged.length > 0) {
          setFolders(merged);
        }
      } catch (error) {
        console.error('Failed to load folders for search filters:', error);
      }
    };

    void loadFoldersIfNeeded();

    return () => {
      cancelled = true;
    };
  }, [accounts, folders.length, setFolders]);

  // 同步到全局状态和触发搜索
  useEffect(() => {
    console.log('🔧 [SearchFilters] useEffect 触发:', {
      isInitialMount: isInitialMount.current,
      localFilters,
      timestamp: new Date().toISOString(),
    });

    // 跳过初始挂载时的调用，避免不必要的搜索
    if (isInitialMount.current) {
      console.log('🔧 [SearchFilters] 跳过初始挂载');
      isInitialMount.current = false;
      return;
    }

    const filters = {
      account_id: localFilters.account_ids.length === 1 ? localFilters.account_ids[0] : undefined,
      folder_id: localFilters.folder_ids.length === 1 ? localFilters.folder_ids[0] : undefined,
      is_read: localFilters.is_read,
      is_starred: localFilters.is_starred,
      is_important: localFilters.is_important,
      has_attachment: localFilters.has_attachment,
      // 确保日期格式为RFC3339格式
      since: localFilters.date_range.start
        ? formatDateToRFC3339(localFilters.date_range.start)
        : undefined,
      before: localFilters.date_range.end
        ? formatDateToRFC3339(localFilters.date_range.end)
        : undefined,
    };

    console.log('🔧 [SearchFilters] 构建的筛选条件:', filters);

    // 只传递有值的筛选条件
    const cleanFilters = Object.fromEntries(
      Object.entries(filters).filter(([, value]) => value !== undefined && value !== null && value !== '')
    );

    console.log('🔧 [SearchFilters] 清理后的筛选条件:', cleanFilters);

    // 使用ref中存储的最新回调函数，避免依赖导致的无限循环
    console.log('🔧 [SearchFilters] 调用 onFiltersChange');
    onFiltersChangeRef.current(cleanFilters);
  }, [localFilters]); // 只依赖localFilters，避免onFiltersChange引起的循环

  // 格式化日期为RFC3339格式
  const formatDateToRFC3339 = (dateStr: string): string => {
    try {
      const date = new Date(dateStr);
      return date.toISOString();
    } catch {
      return dateStr; // 如果格式化失败，返回原始字符串
    }
  };

  // 邮箱账户选项
  const accountOptions = accounts.map((account) => ({
    id: account.id.toString(),
    label: `${account.name} (${account.email})`,
    count: account.total_emails,
    checked: localFilters.account_ids.includes(account.id),
  }));

  // 文件夹选项（按账户分组）
  const folderOptions = folders.map((folder) => ({
    id: folder.id.toString(),
    label: `${folder.display_name || folder.name}`,
    count: folder.total_emails,
    checked: localFilters.folder_ids.includes(folder.id),
  }));

  // 状态选项
  const statusOptions = [
    {
      id: 'unread',
      label: '未读邮件',
      checked: localFilters.is_read === false,
    },
    {
      id: 'read',
      label: '已读邮件',
      checked: localFilters.is_read === true,
    },
    {
      id: 'starred',
      label: '星标邮件',
      checked: localFilters.is_starred === true,
    },
    {
      id: 'important',
      label: '重要邮件',
      checked: localFilters.is_important === true,
    },
    {
      id: 'has_attachment',
      label: '有附件',
      checked: localFilters.has_attachment === true,
    },
  ];

  // 处理账户筛选
  const handleAccountChange = (accountId: string, checked: boolean) => {
    const id = parseInt(accountId);
    setLocalFilters((prev) => ({
      ...prev,
      account_ids: checked
        ? [...prev.account_ids, id]
        : prev.account_ids.filter((aid) => aid !== id),
    }));
  };

  // 处理文件夹筛选
  const handleFolderChange = (folderId: string, checked: boolean) => {
    const id = parseInt(folderId);
    setLocalFilters((prev) => ({
      ...prev,
      folder_ids: checked ? [...prev.folder_ids, id] : prev.folder_ids.filter((fid) => fid !== id),
    }));
  };

  // 处理状态筛选
  const handleStatusChange = (statusId: string, checked: boolean) => {
    setLocalFilters((prev) => {
      const newFilters = { ...prev };

      switch (statusId) {
        case 'unread':
          newFilters.is_read = checked ? false : undefined;
          break;
        case 'read':
          newFilters.is_read = checked ? true : undefined;
          break;
        case 'starred':
          newFilters.is_starred = checked ? true : undefined;
          break;
        case 'important':
          newFilters.is_important = checked ? true : undefined;
          break;
        case 'has_attachment':
          newFilters.has_attachment = checked ? true : undefined;
          break;
      }

      return newFilters;
    });
  };

  // 处理日期范围筛选
  const handleDateRangeChange = (range: { start?: string; end?: string }) => {
    setLocalFilters((prev) => ({
      ...prev,
      date_range: range,
    }));
  };

  // 清除所有筛选
  const clearAllFilters = () => {
    setLocalFilters({
      account_ids: [],
      folder_ids: [],
      is_read: undefined,
      is_starred: undefined,
      is_important: undefined,
      has_attachment: undefined,
      date_range: { start: undefined, end: undefined },
    });
  };

  // 清除账户筛选
  const clearAccountFilters = () => {
    setLocalFilters((prev) => ({ ...prev, account_ids: [] }));
  };

  // 清除文件夹筛选
  const clearFolderFilters = () => {
    setLocalFilters((prev) => ({ ...prev, folder_ids: [] }));
  };

  // 清除状态筛选
  const clearStatusFilters = () => {
    setLocalFilters((prev) => ({
      ...prev,
      is_read: undefined,
      is_starred: undefined,
      is_important: undefined,
      has_attachment: undefined,
    }));
  };

  // 清除日期筛选
  const clearDateFilters = () => {
    setLocalFilters((prev) => ({ ...prev, date_range: { start: undefined, end: undefined } }));
  };

  // 检查是否有活动筛选
  const hasActiveFilters =
    localFilters.account_ids.length > 0 ||
    localFilters.folder_ids.length > 0 ||
    localFilters.is_read !== undefined ||
    localFilters.is_starred !== undefined ||
    localFilters.is_important !== undefined ||
    localFilters.has_attachment !== undefined ||
    localFilters.date_range.start ||
    localFilters.date_range.end;

  return (
    <div className="w-full h-full bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 overflow-y-auto">
      {/* 筛选标题和清除按钮 */}
      <div className="p-4 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">筛选条件</h3>
          {hasActiveFilters && (
            <Button
              variant="ghost"
              size="sm"
              onClick={clearAllFilters}
              className="text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300"
            >
              <X className="w-4 h-4 mr-1" />
              清除全部
            </Button>
          )}
        </div>
      </div>

      {/* 筛选选项 */}
      <div className="p-4 space-y-0">
        {/* 邮箱账户筛选 */}
        {accountOptions.length > 0 && (
          <FilterGroup
            title="邮箱账户"
            options={accountOptions}
            onOptionChange={handleAccountChange}
            onClearAll={clearAccountFilters}
          />
        )}

        {/* 文件夹筛选 */}
        {folderOptions.length > 0 && (
          <FilterGroup
            title="文件夹"
            options={folderOptions}
            onOptionChange={handleFolderChange}
            onClearAll={clearFolderFilters}
          />
        )}

        {/* 状态筛选 */}
        <FilterGroup
          title="邮件状态"
          options={statusOptions}
          onOptionChange={handleStatusChange}
          onClearAll={clearStatusFilters}
        />

        {/* 日期范围筛选 */}
        <DateRangeFilter
          title="时间范围"
          value={localFilters.date_range}
          onChange={handleDateRangeChange}
          onClear={clearDateFilters}
        />
      </div>
    </div>
  );
}
