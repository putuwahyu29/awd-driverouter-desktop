export interface FileRecord {
  id: string;
  name: string;
  size: number;
  isFolder: boolean;
  parentId: string;
  provider: string;
  accountId: string;
  physicalId: string;
  createdAt: string;
  modifiedAt: string;
  starred: boolean;
  shared: boolean;
}

export interface AccountRecord {
  id: string;
  provider: string;
  displayName: string;
  email: string;
  usedSpace: number;
  totalSpace: number;
  active: boolean;
}

export interface TransferItem {
  id: string;
  name: string;
  type: 'upload' | 'download' | 'transfer';
  status: 'started' | 'uploading' | 'downloading' | 'transferring' | 'completed' | 'failed';
  progress?: number;
  error?: string;
}

export type AppDialog =
  | {
      type: 'info';
      variant?: 'info' | 'warning' | 'danger';
      title: string;
      message: string;
      confirmLabel?: string;
    }
  | {
      type: 'confirm';
      variant?: 'warning' | 'danger';
      title: string;
      message: string;
      confirmLabel?: string;
      cancelLabel?: string;
      onConfirm: () => void | Promise<void>;
    }
  | {
      type: 'prompt';
      variant?: 'info' | 'warning';
      title: string;
      message: string;
      inputLabel: string;
      defaultValue: string;
      confirmLabel?: string;
      cancelLabel?: string;
      onConfirm: (value: string) => void | Promise<void>;
    };
