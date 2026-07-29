import React, { useState, useEffect, useRef, useMemo } from 'react';
import logoImg from './assets/images/logo.png';
import {
  GetFiles,
  GetAccounts,
  DisconnectAccount,
  GetSettings,
  SaveSetting,
  SaveCredentials,
  StartOAuthFlow,
  QuitApp,
  AddWebDAVAccount,
  AddS3Account,
  AddTelegramAccount,
  SendTelegramCode,
  VerifyTelegramCode,
  CreateFolder,
  RenameFile,
  UploadFileDialog,
  DeleteFile,
  GetTrashedFiles,
  RestoreFile,
  PermanentlyDeleteFile,
  DownloadFileDialog,
  DownloadBulkDialog,
  ToggleStarred,
  SyncDrives,
  GetRecentFiles,
  PreviewFile,
  UploadFileFromPath,
  CopyFileToAccount,
  GetVirtualFolders,
  RemoteUploadFromURL,
  CompressFilesToZip,
  ExtractZipFile,
  FindDuplicateFiles,
  GetFileWebURL,
  OpenFileInBrowser,
  GetFilePermissions,
  AddFilePermission,
  DeleteFilePermission,
  SetFileGeneralAccess,
  GetFileActivities,
  GetGeneralActivities,
  CreateWebShare,
  ToggleAccountActive
} from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

import { FileRecord, AccountRecord, TransferItem, AppDialog } from './types';
import { TRANSLATIONS } from './translations';
import {
  IconHome,
  IconFolder,
  IconFile,
  IconStar,
  IconSettings,
  IconCloud,
  IconSearch,
  IconPlus,
  IconDots,
  IconDelete,
  IconRename,
  IconDownload,
  IconInfo,
  IconChevronRight,
  IconClose,
  IconRefresh,
  IconWarning,
  IconEye,
  IconEyeOff,
  IconGoogleDrive,
  IconOneDrive,
  IconDropbox,
  IconBox,
  IconYandex,
  IconPCloud,
  IconMega,
  IconKoofr,
  IconMediaFire,
  Icon4Shared,
  IconB2,
  IconSmb,
  IconFtp,
  IconSftp,
  IconTelegram,
  FolderIconWithShared,
  FileIcon
} from './components/Icons';
import AboutView from './components/AboutView';
import VirtualDriveView from './components/VirtualDriveView';
import WebShareManagement from './components/WebShareManagement';
import { PROVIDER_GUIDES } from './providerGuides';

// @ts-ignore
import { CheckForUpdates, OpenReleaseURL } from '../wailsjs/go/main/App';

function App() {
  const [view, setView] = useState<'home' | 'explorer' | 'starred' | 'accounts' | 'settings' | 'shared' | 'recent' | 'storage' | 'trash' | 'about' | 'rclone' | 'webshare'>('home');
  const [storageTab, setStorageTab] = useState<'overview' | 'allocation' | 'duplicates'>('overview');
  const [settingsTab, setSettingsTab] = useState<'general' | 'vdisk' | 'backup' | 'api'>('general');
  const [theme, setTheme] = useState<'light' | 'dark'>('dark');
  const [files, setFiles] = useState<FileRecord[]>([]);
  const [parentID, setParentID] = useState<string>('root');
  const [breadcrumbs, setBreadcrumbs] = useState<{ id: string; name: string }[]>([{ id: 'root', name: 'My Drive' }]);
  
  // Filtering and Searching
  const [searchKeyword, setSearchKeyword] = useState<string>('');
  const [localSearchTerm, setLocalSearchTerm] = useState<string>('');

  // Sync external changes of searchKeyword back to localSearchTerm (e.g. resets)
  useEffect(() => {
    if (searchKeyword !== localSearchTerm) {
      setLocalSearchTerm(searchKeyword);
    }
  }, [searchKeyword]);

  // Debounce updating searchKeyword from localSearchTerm
  useEffect(() => {
    const timer = setTimeout(() => {
      if (localSearchTerm !== searchKeyword) {
        setSearchKeyword(localSearchTerm);
      }
    }, 300);
    return () => clearTimeout(timer);
  }, [localSearchTerm, searchKeyword]);
  
  // Selection details
  const [selectedFile, setSelectedFile] = useState<FileRecord | null>(null);
  const [selectedIDs, setSelectedIDs] = useState<string[]>([]);
  const [detailsSidebar, setDetailsSidebar] = useState<boolean>(false);
  const [activeLayout, setActiveLayout] = useState<'grid' | 'list'>('list');
  
  const [accounts, setAccounts] = useState<AccountRecord[]>([]);
  const [settings, setSettings] = useState<Record<string, string>>({});
  
  // File transfer states
  const [transferFile, setTransferFile] = useState<FileRecord | null>(null);
  const [virtualFolders, setVirtualFolders] = useState<FileRecord[]>([]);
  const [selectedDestAccountID, setSelectedDestAccountID] = useState<string>('');
  const [selectedDestFolderID, setSelectedDestFolderID] = useState<string>('root');
  const [transferLoading, setTransferLoading] = useState<boolean>(false);
  
  // Advanced Search Filter States
  const [filterAccountID, setFilterAccountID] = useState<string>('all');
  const [filterFileType, setFilterFileType] = useState<string>('all');
  const [filterFileSize, setFilterFileSize] = useState<string>('all');
  const [filterModDate, setFilterModDate] = useState<string>('all');
  const [advancedSearchOpen, setAdvancedSearchOpen] = useState<boolean>(false);

  // Recents filter state
  const [recentAccountFilter, setRecentAccountFilter] = useState<string>('all');

  // Web Share modal state
  const [createShareFile, setCreateShareFile] = useState<FileRecord | null>(null);
  const [sharePassword, setSharePassword] = useState<string>('');
  const [creatingShare, setCreatingShare] = useState<boolean>(false);

  const openWebShareModal = (file: FileRecord) => {
    setCreateShareFile(file);
    setSharePassword('');
  };

  const handleCreateWebShareSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!createShareFile) return;
    setCreatingShare(true);
    try {
      await CreateWebShare(createShareFile.id, sharePassword);
      showToast(lang === 'id' ? 'Web share link berhasil dibuat!' : 'Web share link created successfully!');
      setCreateShareFile(null);
      setSharePassword('');
      setView('webshare');
    } catch (err) {
      showInfoDialog("Error", "Failed to create web share link: " + err);
    } finally {
      setCreatingShare(false);
    }
  };

  // Local Settings Options
  const [lang, setLang] = useState<'en' | 'id'>('en');
  const [minToTray, setMinToTray] = useState<boolean>(true);
  const [autoStartup, setAutoStartup] = useState<boolean>(false);
  const [backupInterval, setBackupInterval] = useState<number>(60);
  const langRef = useRef<'en' | 'id'>('en');

  // Toast notification state
  const [toast, setToast] = useState<{ message: string; visible: boolean }>({ message: '', visible: false });
  const toastTimeoutRef = useRef<any>(null);

  const showToast = (message: string) => {
    if (toastTimeoutRef.current) {
      clearTimeout(toastTimeoutRef.current);
    }
    setToast({ message, visible: true });
    toastTimeoutRef.current = setTimeout(() => {
      setToast({ message: '', visible: false });
    }, 3000);
  };

  // Account email visibility state (masked by default)
  const [showEmails, setShowEmails] = useState<{ [accountId: string]: boolean }>({});
  const toggleShowEmail = (accId: string) => {
    setShowEmails(prev => ({ ...prev, [accId]: !prev[accId] }));
  };

  const maskEmail = (email: string) => {
    if (!email) return '';
    if (email.includes('@')) {
      const [user, domain] = email.split('@');
      if (user.length <= 2) {
        return `${user[0] || '*'}***@${domain}`;
      }
      return `${user[0]}***${user[user.length - 1]}@${domain}`;
    }
    if (email.length > 5) {
      return email.substring(0, 3) + '***' + email.substring(email.length - 2);
    }
    return '••••••••';
  };

  // Share Modal States
  const [shareFile, setShareFile] = useState<FileRecord | null>(null);
  const [sharePermissions, setSharePermissions] = useState<any[]>([]);
  const [shareLoading, setShareLoading] = useState<boolean>(false);
  const [shareEmail, setShareEmail] = useState<string>('');
  const [shareRole, setShareRole] = useState<'reader' | 'writer'>('reader');
  const [shareGeneralAccess, setShareGeneralAccess] = useState<string>('restricted');
  const [shareGeneralRole, setShareGeneralRole] = useState<'reader' | 'writer'>('reader');

  // Activity Log States
  const [detailsTab, setDetailsTab] = useState<'details' | 'activity'>('details');
  const [fileActivities, setFileActivities] = useState<any[]>([]);
  const [activitiesLoading, setActivitiesLoading] = useState<boolean>(false);
  const [generalActivities, setGeneralActivities] = useState<any[]>([]);
  const [generalActivitiesLoading, setGeneralActivitiesLoading] = useState<boolean>(false);

  // Backup Sync Task States
  const [syncTasks, setSyncTasks] = useState<any[]>([]);
  const [syncTasksLoading, setSyncTasksLoading] = useState<boolean>(false);
  const [backupLocalPath, setBackupLocalPath] = useState<string>('');
  const [backupTargetFolderID, setBackupTargetFolderID] = useState<string>('root');
  const [backupAccountID, setBackupAccountID] = useState<string>('auto');
  const [backupSyncMode, setBackupSyncMode] = useState<string>('one-way');
  const [editingSyncTask, setEditingSyncTask] = useState<any | null>(null);

  // Remote URL upload state
  const [remoteUploadURL, setRemoteUploadURL] = useState<string>('');
  const [remoteUploadAccountID, setRemoteUploadAccountID] = useState<string>('');

  // ZIP archive state
  const [zipArchiveName, setZipArchiveName] = useState<string>('');

  // Duplicates finder state
  const [duplicateFiles, setDuplicateFiles] = useState<FileRecord[]>([]);
  const [duplicatesLoading, setDuplicatesLoading] = useState<boolean>(false);

  // Auto Update check state
  const [appVersion, setAppVersion] = useState<string>('1.0.0');
  const [updateInfo, setUpdateInfo] = useState<{
    has_update: boolean;
    latest_version: string;
    update_url: string;
    release_notes: string;
  } | null>(null);
  const [showUpdateModal, setShowUpdateModal] = useState<boolean>(false);

  // Modals state
  const [modal, setModal] = useState<{ type: 'create-folder' | 'add-account' | 'credentials' | 'mega' | 'koofr' | 'mediafire' | 'fourshared' | 'b2' | 'smb' | 'ftp' | 'sftp' | 'webdav' | 's3' | 'telegram' | 'telegram_user' | 'transfer-file' | 'remote-upload' | 'compress-zip' | 'share' | 'backup-task' | null; provider?: string } | null>(null);
  const [actionDialog, setActionDialog] = useState<AppDialog | null>(null);
  const [dialogInput, setDialogInput] = useState<string>('');
  const [folderNameInput, setFolderNameInput] = useState<string>('');
  const [credClientID, setCredClientID] = useState<string>('');
  const [credClientSecret, setCredClientSecret] = useState<string>('');
  
  // MEGA input states
  const [megaEmail, setMegaEmail] = useState<string>('');
  const [megaPassword, setMegaPassword] = useState<string>('');
  const [megaLoading, setMegaLoading] = useState<boolean>(false);
  const [megaError, setMegaError] = useState<string>('');

  // 4Shared input states
  const [foursharedEmail, setFoursharedEmail] = useState<string>('');
  const [foursharedPassword, setFoursharedPassword] = useState<string>('');
  const [foursharedLoading, setFoursharedLoading] = useState<boolean>(false);
  const [foursharedError, setFoursharedError] = useState<string>('');

  // Backblaze B2 input states
  const [b2DisplayName, setB2DisplayName] = useState<string>('');
  const [b2KeyID, setB2KeyID] = useState<string>('');
  const [b2AppKey, setB2AppKey] = useState<string>('');
  const [b2Bucket, setB2Bucket] = useState<string>('');
  const [b2Loading, setB2Loading] = useState<boolean>(false);
  const [b2Error, setB2Error] = useState<string>('');

  // SMB input states
  const [smbDisplayName, setSmbDisplayName] = useState<string>('');
  const [smbHost, setSmbHost] = useState<string>('');
  const [smbShare, setSmbShare] = useState<string>('');
  const [smbUsername, setSmbUsername] = useState<string>('');
  const [smbPassword, setSmbPassword] = useState<string>('');
  const [smbLoading, setSmbLoading] = useState<boolean>(false);
  const [smbError, setSmbError] = useState<string>('');

  // Koofr input states
  const [koofrUser, setKoofrUser] = useState<string>('');
  const [koofrPass, setKoofrPass] = useState<string>('');
  const [koofrLoading, setKoofrLoading] = useState<boolean>(false);
  const [koofrError, setKoofrError] = useState<string>('');

  // MediaFire input states
  const [mediafireEmail, setMediafireEmail] = useState<string>('');
  const [mediafirePassword, setMediafirePassword] = useState<string>('');
  const [mediafireLoading, setMediafireLoading] = useState<boolean>(false);
  const [mediafireError, setMediafireError] = useState<string>('');

  // FTP / SFTP input states
  const [serverDisplayName, setServerDisplayName] = useState<string>('');
  const [serverHost, setServerHost] = useState<string>('');
  const [serverPort, setServerPort] = useState<number>(21);
  const [serverUsername, setServerUsername] = useState<string>('');
  const [serverPassword, setServerPassword] = useState<string>('');
  const [serverBaseDir, setServerBaseDir] = useState<string>('/');
  const [serverLoading, setServerLoading] = useState<boolean>(false);
  const [serverError, setServerError] = useState<string>('');

  // WebDAV input states
  const [webdavName, setWebdavName] = useState<string>('');
  const [webdavUrl, setWebdavUrl] = useState<string>('');
  const [webdavUsername, setWebdavUsername] = useState<string>('');
  const [webdavPassword, setWebdavPassword] = useState<string>('');
  const [webdavLoading, setWebdavLoading] = useState<boolean>(false);
  const [webdavError, setWebdavError] = useState<string>('');

  // S3 input states
  const [s3Name, setS3Name] = useState<string>('');
  const [s3Endpoint, setS3Endpoint] = useState<string>('');
  const [s3Bucket, setS3Bucket] = useState<string>('');
  const [s3AccessKey, setS3AccessKey] = useState<string>('');
  const [s3SecretKey, setS3SecretKey] = useState<string>('');
  const [s3Loading, setS3Loading] = useState<boolean>(false);
  const [s3Error, setS3Error] = useState<string>('');

  // Telegram Bot input states
  const [telegramName, setTelegramName] = useState<string>('');
  const [telegramToken, setTelegramToken] = useState<string>('');
  const [telegramChatID, setTelegramChatID] = useState<string>('');
  const [telegramLoading, setTelegramLoading] = useState<boolean>(false);
  const [telegramError, setTelegramError] = useState<string>('');

  // Telegram User (MTProto) input states
  const [tgUserDisplayName, setTgUserDisplayName] = useState<string>('');
  const [tgUserPhone, setTgUserPhone] = useState<string>('');
  const [tgUserCode, setTgUserCode] = useState<string>('');
  const [tgUserPassword, setTgUserPassword] = useState<string>('');
  const [tgUserStep, setTgUserStep] = useState<'phone' | 'code' | 'password'>('phone');
  const [tgUserLoading, setTgUserLoading] = useState<boolean>(false);
  const [tgUserError, setTgUserError] = useState<string>('');
  
  // Floating context menus
  const [contextMenu, setContextMenu] = useState<{ visible: boolean; x: number; y: number; file: FileRecord | null }>({
    visible: false,
    x: 0,
    y: 0,
    file: null
  });

  // Preview modal states
  const [previewFile, setPreviewFile] = useState<FileRecord | null>(null);
  const [previewData, setPreviewData] = useState<any>(null);
  const [previewLoading, setPreviewLoading] = useState<boolean>(false);
  const [previewError, setPreviewError] = useState<string>('');
  
  // Transfer status drawer
  const [transfers, setTransfers] = useState<TransferItem[]>([]);
  const [syncing, setSyncing] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(false);

  // New Dropdown state
  const [showNewDropdown, setShowNewDropdown] = useState<boolean>(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState<boolean>(false);
  const newDropdownRef = useRef<HTMLDivElement>(null);

  // Guide collapses in settings
  const [activeGuide, setActiveGuide] = useState<string | null>(null);
  const [showGuideModal, setShowGuideModal] = useState<boolean>(false);

  // Translate helper
  const t = (key: string): string => {
    return TRANSLATIONS[lang]?.[key] || TRANSLATIONS['en']?.[key] || key;
  };

  useEffect(() => {
    langRef.current = lang;
  }, [lang]);

  const translate = (key: string): string => {
    return TRANSLATIONS[langRef.current]?.[key] || TRANSLATIONS['en']?.[key] || key;
  };

  const closeActionDialog = () => {
    setActionDialog(null);
    setDialogInput('');
  };

  const showInfoDialog = (title: string, message: string, variant: 'info' | 'warning' | 'danger' = 'danger', confirmLabel?: string) => {
    setActionDialog({ type: 'info', title, message, variant, confirmLabel });
  };

  const showConfirmDialog = (
    title: string,
    message: string,
    onConfirm: () => void | Promise<void>,
    options?: { variant?: 'warning' | 'danger'; confirmLabel?: string; cancelLabel?: string },
  ) => {
    setActionDialog({
      type: 'confirm',
      title,
      message,
      onConfirm,
      variant: options?.variant ?? 'warning',
      confirmLabel: options?.confirmLabel,
      cancelLabel: options?.cancelLabel,
    });
  };

  const showPromptDialog = (
    title: string,
    message: string,
    inputLabel: string,
    defaultValue: string,
    onConfirm: (value: string) => void | Promise<void>,
    options?: { variant?: 'info' | 'warning'; confirmLabel?: string; cancelLabel?: string },
  ) => {
    setDialogInput(defaultValue);
    setActionDialog({
      type: 'prompt',
      title,
      message,
      inputLabel,
      defaultValue,
      onConfirm,
      variant: options?.variant ?? 'info',
      confirmLabel: options?.confirmLabel,
      cancelLabel: options?.cancelLabel,
    });
  };

  const isProviderConfigured = (providerName: string): boolean => {
    if (['google', 'onedrive', 'dropbox', 'box', 'yandex', 'pcloud'].includes(providerName)) {
      const cid = settings[`${providerName}_client_id`]?.trim();
      const secret = settings[`${providerName}_client_secret`]?.trim();
      return !!(cid && secret);
    }
    if (providerName === 'telegram_user') {
      const apiId = settings['telegram_api_id']?.trim();
      const apiHash = settings['telegram_api_hash']?.trim();
      return !!(apiId && apiHash);
    }
    return true;
  };

  // Hook context menu close and Wails events
  useEffect(() => {
    // Close context menu on click anywhere
    const closeMenu = () => setContextMenu(prev => ({ ...prev, visible: false }));
    window.addEventListener('click', closeMenu);

    // Prevent default browser context menu globally
    const preventDefaultMenu = (e: MouseEvent) => e.preventDefault();
    window.addEventListener('contextmenu', preventDefaultMenu);

    // Close new dropdown on click outside
    const handleOutsideClick = (e: MouseEvent) => {
      if (newDropdownRef.current && !newDropdownRef.current.contains(e.target as Node)) {
        setShowNewDropdown(false);
      }
    };
    window.addEventListener('mousedown', handleOutsideClick);
    
    // Load initial accounts and settings
    fetchAccounts();
    fetchSettings();

    // Check if auto startup is enabled in OS
    // @ts-ignore
    window.go?.main?.App?.IsStartupEnabled?.()?.then((enabled: boolean) => {
      setAutoStartup(!!enabled);
    });
    
    // Setup background theme
    const localTheme = localStorage.getItem('driverouter-theme') as 'light' | 'dark' | null;
    if (localTheme) {
      setTheme(localTheme);
      document.body.setAttribute('data-theme', localTheme);
    } else {
      document.body.setAttribute('data-theme', 'dark');
    }

    // Connect Wails background events
    const offUploadStart = EventsOn('upload_started', (payload: string | { uploadId?: string; filename?: string }) => {
      const filename = typeof payload === 'string' ? payload : payload?.filename || '';
      const uploadId = typeof payload === 'string' ? payload : payload?.uploadId || filename;
      setTransfers(prev => [{ id: uploadId, name: filename, type: 'upload', status: 'started', progress: 0 }, ...prev]);
    });

    const offUploadDone = EventsOn('upload_completed', (payload: string | { uploadId?: string; filename?: string }) => {
      const filename = typeof payload === 'string' ? payload : payload?.filename || '';
      const uploadId = typeof payload === 'string' ? payload : payload?.uploadId || filename;
      setTransfers(prev => prev.map(t => (t.id === uploadId || t.name === filename) ? { ...t, status: 'completed', progress: 100 } : t));
      fetchFiles(parentID);
      fetchAccounts(); // refresh storage quotas
    });

    const offUploadFail = EventsOn('upload_failed', (data: { uploadId?: string; filename: string; error: string }) => {
      setTransfers(prev => prev.map(t => (t.id === data.uploadId || t.name === data.filename) ? { ...t, status: 'failed', error: data.error } : t));
    });

    const offDownloadStart = EventsOn('download_started', (data: { downloadId: string; filename: string }) => {
      setTransfers(prev => [{ id: data.downloadId, name: data.filename, type: 'download', status: 'started', progress: 0 }, ...prev]);
    });

    const offDownloadDone = EventsOn('download_completed', (data: { downloadId: string; filename: string }) => {
      setTransfers(prev => prev.map(t => t.id === data.downloadId ? { ...t, status: 'completed', progress: 100 } : t));
    });

    const offDownloadFail = EventsOn('download_failed', (data: { downloadId: string; filename: string; error: string }) => {
      setTransfers(prev => prev.map(t => t.id === data.downloadId ? { ...t, status: 'failed', error: data.error } : t));
    });

    const offTransferStart = EventsOn('transfer_started', (data: { transferId: string; filename: string }) => {
      setTransfers(prev => [{ id: data.transferId, name: data.filename, type: 'transfer', status: 'started', progress: 0 }, ...prev]);
    });

    const offTransferDone = EventsOn('transfer_completed', (data: { transferId: string; filename: string }) => {
      setTransfers(prev => prev.map(t => t.id === data.transferId ? { ...t, status: 'completed', progress: 100 } : t));
      fetchFiles(parentID);
      fetchAccounts();
    });

    const offTransferFail = EventsOn('transfer_failed', (data: { transferId: string; filename: string; error: string }) => {
      setTransfers(prev => prev.map(t => t.id === data.transferId ? { ...t, status: 'failed', error: data.error } : t));
    });

    const offExitPrompt = EventsOn('app:request-exit-confirm', () => {
      setActionDialog({
        type: 'confirm',
        variant: 'warning',
        title: translate('confirmExitTitle'),
        message: translate('confirmExitMessage'),
        confirmLabel: translate('confirmExitButton'),
        cancelLabel: translate('cancel'),
        onConfirm: () => QuitApp(),
      });
    });

    const offMenuNav = EventsOn('menu:navigate', (page: string) => {
      if (page === 'about') {
        setView('about');
      }
    });

    // Get dynamic app version
    // @ts-ignore
    window.go?.main?.App?.GetAppVersion?.()?.then((ver: string) => {
      if (ver) setAppVersion(ver);
    });

    const checkUpdates = () => {
      CheckForUpdates()
        .then((info: any) => {
          if (info && info.has_update) {
            setUpdateInfo(info);
            setShowUpdateModal(true);
          }
        })
        .catch((e: any) => console.error('Failed to check for updates:', e));
    };

    const offCheckUpdates = EventsOn('menu:check-updates', () => {
      checkUpdates();
    });

    // Check for updates on startup and every 2 hours
    checkUpdates();
    const updateInterval = setInterval(checkUpdates, 2 * 60 * 60 * 1000);

    const uploadSocket = new WebSocket('ws://127.0.0.1:5999/ws/uploads');
    uploadSocket.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data) as { type?: string; id?: string; filename?: string; percent?: number; error?: string; size?: number; transferType?: string };
        
        if (message.type === 'upload_progress') {
          setTransfers(prev => prev.map(t => (
            t.id === message.id || t.name === message.filename
              ? { ...t, status: message.error ? 'failed' : 'uploading', progress: message.percent ?? t.progress, error: message.error }
              : t
          )));
        } else if (message.type === 'download_progress') {
          setTransfers(prev => prev.map(t => (
            t.id === message.id
              ? { ...t, status: message.error ? 'failed' : 'downloading', progress: message.percent ?? t.progress, error: message.error }
              : t
          )));
        }
      } catch (error) {
        console.error(error);
      }
    };

    const offFileDrop = EventsOn('wails:file-drop', (x: number, y: number, paths: string[]) => {
      if (paths && paths.length > 0) {
        paths.forEach(filePath => {
          UploadFileFromPath(parentID, filePath).catch(e => console.error(e));
        });
      }
    });

    return () => {
      clearInterval(updateInterval);
      window.removeEventListener('click', closeMenu);
      window.removeEventListener('contextmenu', preventDefaultMenu);
      window.removeEventListener('mousedown', handleOutsideClick);
      if (offUploadStart) offUploadStart();
      if (offUploadDone) offUploadDone();
      if (offUploadFail) offUploadFail();
      if (offDownloadStart) offDownloadStart();
      if (offDownloadDone) offDownloadDone();
      if (offDownloadFail) offDownloadFail();
      if (offExitPrompt) offExitPrompt();
      if (offMenuNav) offMenuNav();
      if (offCheckUpdates) offCheckUpdates();
      if (offFileDrop) offFileDrop();
      uploadSocket.close();
    };
  }, [parentID]);

  // Load files whenever parent folder or search queries change
  useEffect(() => {
    setSelectedIDs([]);
    setSelectedFile(null);
    setFiles([]); // Clear previous view data immediately on menu switch

    if (view === 'explorer') {
      fetchFiles(parentID, true);

      // Realtime listener when background sync updates database files
      const offFilesUpdated = EventsOn('files-updated', (updatedParentID: string) => {
        if (updatedParentID === parentID) {
          fetchFiles(parentID, false);
        }
      });

      const interval = setInterval(() => {
        GetFiles(parentID, false, searchKeyword).then(list => {
          setFiles(list || []);
        }).catch(e => console.error(e));
      }, 10000); // Background refresh every 10 seconds

      return () => {
        if (offFilesUpdated) offFilesUpdated();
        clearInterval(interval);
      };
    } else if (view === 'starred') {
      fetchStarredFiles(true);
    } else if (view === 'recent') {
      fetchRecentFiles(true);
    } else if (view === 'shared') {
      fetchFiles('__shared__', true);
    } else if (view === 'trash') {
      fetchTrashedFiles(true);
    } else if (view === 'home') {
      fetchGeneralActivities();
    } else if (view === 'settings') {
      fetchSyncTasks();
      const offSync = EventsOn('sync_tasks_updated', () => {
        fetchSyncTasks();
      });
      return () => {
        if (offSync) offSync();
      };
    }
  }, [parentID, view, searchKeyword]);

  // Advanced Search & Filter computation
  const filteredFiles = useMemo(() => {
    return files.filter(f => {
      // 1. Filter by Cloud Provider Account ID
      if (filterAccountID !== 'all') {
        if (f.accountId !== filterAccountID) {
          return false;
        }
      }

      // 2. Filter by File Category Type
      if (filterFileType !== 'all') {
        if (f.isFolder) {
          return false; // Type filter only applies to files
        }
        const ext = f.name.substring(f.name.lastIndexOf('.')).toLowerCase();
        const isImg = ['.jpg', '.jpeg', '.png', '.gif', '.webp', '.svg', '.bmp', '.ico'].includes(ext);
        const isVid = ['.mp4', '.webm', '.mov', '.mkv', '.avi'].includes(ext);
        const isAud = ['.mp3', '.wav', '.flac', '.aac', '.m4a', '.ogg'].includes(ext);
        const isDoc = ['.pdf', '.doc', '.docx', '.txt', '.rtf', '.xls', '.xlsx', '.csv', '.ppt', '.pptx'].includes(ext);

        if (filterFileType === 'image' && !isImg) return false;
        if (filterFileType === 'video' && !isVid) return false;
        if (filterFileType === 'audio' && !isAud) return false;
        if (filterFileType === 'document' && !isDoc) return false;
        if (filterFileType === 'other' && (isImg || isVid || isAud || isDoc)) return false;
      }

      // 3. Filter by File Size
      if (filterFileSize !== 'all') {
        if (f.isFolder) return false;
        const sizeMB = f.size / (1024 * 1024);
        if (filterFileSize === 'gt10mb' && sizeMB <= 10) return false;
        if (filterFileSize === 'gt100mb' && sizeMB <= 100) return false;
        if (filterFileSize === 'gt1gb' && sizeMB <= 1024) return false;
      }

      // 4. Filter by Modified Date
      if (filterModDate !== 'all') {
        const modTime = new Date(f.modifiedAt).getTime();
        const now = new Date().getTime();
        const diffHrs = (now - modTime) / (1000 * 60 * 60);
        const diffDays = diffHrs / 24;

        if (filterModDate === 'today' && diffHrs > 24) return false;
        if (filterModDate === 'yesterday' && (diffHrs <= 24 || diffHrs > 48)) return false;
        if (filterModDate === 'week' && diffDays > 7) return false;
        if (filterModDate === 'month' && diffDays > 30) return false;
      }

      return true;
    });
  }, [files, filterAccountID, filterFileType, filterFileSize, filterModDate]);

  // Chronological aggregator for Recent files
  const groupRecentFiles = (recentList: FileRecord[]) => {
    const today: FileRecord[] = [];
    const yesterday: FileRecord[] = [];
    const lastWeek: FileRecord[] = [];
    const older: FileRecord[] = [];

    const now = new Date().getTime();

    recentList.forEach(f => {
      const time = new Date(f.modifiedAt).getTime();
      const diffHours = (now - time) / (1000 * 60 * 60);
      const diffDays = diffHours / 24;

      if (diffHours <= 24) {
        today.push(f);
      } else if (diffHours <= 48) {
        yesterday.push(f);
      } else if (diffDays <= 7) {
        lastWeek.push(f);
      } else {
        older.push(f);
      }
    });

    return [
      { label: "Today", items: today },
      { label: "Yesterday", items: yesterday },
      { label: "Last 7 Days", items: lastWeek },
      { label: "Older", items: older }
    ].filter(g => g.items.length > 0);
  };

  const getFilteredRecentFiles = () => {
    if (recentAccountFilter === 'all') return filteredFiles;
    return filteredFiles.filter(f => f.accountId === recentAccountFilter);
  };

  useEffect(() => {
    if (storageTab === 'duplicates') {
      scanDuplicateFiles();
    }
  }, [storageTab]);

  useEffect(() => {
    if (selectedFile && detailsTab === 'activity' && detailsSidebar) {
      fetchFileActivities(selectedFile.id);
    }
  }, [selectedFile, detailsTab, detailsSidebar]);

  const fetchRecentFiles = async (showFullLoading: boolean = true) => {
    if (showFullLoading) setLoading(true);
    try {
      const list = await GetRecentFiles();
      setFiles(list || []);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const formatDateTime = (dateStr: string) => {
    try {
      const d = new Date(dateStr);
      return d.toLocaleString(lang === 'id' ? 'id-ID' : 'en-US', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false
      });
    } catch (e) {
      return dateStr;
    }
  };

  const translateActivityDetails = (details: string) => {
    if (!details) return '';
    if (lang === 'en') return details;

    let result = details;
    result = result.replace(/^Moved '(.+)' to trash$/i, "Memindahkan '$1' ke keranjang sampah");
    result = result.replace(/^Permanently deleted '(.+)'$/i, "Menghapus secara permanen '$1'");
    result = result.replace(/^Restored '(.+)' from trash$/i, "Memulihkan '$1' dari keranjang sampah");
    result = result.replace(/^Renamed '(.+)' to '(.+)'$/i, "Mengubah nama '$1' menjadi '$2'");
    result = result.replace(/^Uploaded '(.+)'$/i, "Mengunggah '$1'");
    result = result.replace(/^Downloaded '(.+)'$/i, "Mengunduh '$1'");
    result = result.replace(/^Created folder '(.+)'$/i, "Membuat folder '$1'");
    result = result.replace(/^Compressed (.+) files to ZIP '(.+)'$/i, "Mengompres $1 file ke ZIP '$2'");
    result = result.replace(/^Extracted ZIP '(.+)'$/i, "Mengekstrak ZIP '$1'");
    result = result.replace(/^Copied '(.+)' to (.+) account$/i, "Menyalin '$1' ke akun $2");
    result = result.replace(/^Changed general access to restricted$/i, "Mengubah akses umum menjadi dibatasi");
    result = result.replace(/^Changed general access to anyone (.+)$/i, "Mengubah akses umum menjadi siapa saja dengan link ($1)");
    result = result.replace(/^Added permission (.+) with role (.+)$/i, "Menambahkan izin untuk $1 dengan peran $2");
    result = result.replace(/^Removed permission (.+)$/i, "Menghapus izin untuk $1");
    return result;
  };

  const fetchSyncTasks = async () => {
    setSyncTasksLoading(true);
    try {
      // @ts-ignore
      const list = await window.go.main.App.GetSyncTasks();
      setSyncTasks(list || []);
    } catch (e) {
      console.error(e);
    } finally {
      setSyncTasksLoading(false);
    }
  };

  const handleSelectBackupFolder = async () => {
    try {
      // @ts-ignore
      const res = await window.go.main.App.SelectBackupFolder();
      if (res) {
        setBackupLocalPath(res);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const handleAddSyncTask = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!backupLocalPath) {
      showInfoDialog("Error", "Please select a local folder");
      return;
    }
    try {
      if (editingSyncTask) {
        // @ts-ignore
        await window.go.main.App.UpdateSyncTask(editingSyncTask.id, backupTargetFolderID, backupAccountID, backupSyncMode);
        showToast('Backup task updated successfully!');
      } else {
        // @ts-ignore
        await window.go.main.App.AddSyncTask(backupLocalPath, backupTargetFolderID, backupAccountID, backupSyncMode);
        showToast('Backup task added successfully!');
      }
      setBackupLocalPath('');
      setEditingSyncTask(null);
      setModal(null);
      fetchSyncTasks();
    } catch (err) {
      showInfoDialog("Error", "Failed to save backup task: " + err);
    }
  };

  const handleBackupIntervalChange = async (seconds: number) => {
    setBackupInterval(seconds);
    try {
      // @ts-ignore
      await window.go.main.App.UpdateBackupInterval(seconds);
      showToast(lang === 'id' ? 'Interval backup diperbarui!' : 'Backup interval updated!');
    } catch (e) {
      console.error(e);
      showInfoDialog("Error", "Failed to update backup interval: " + e);
    }
  };

  const handleEditSyncTaskClick = (task: any) => {
    setEditingSyncTask(task);
    setBackupLocalPath(task.localPath);
    setBackupTargetFolderID(task.targetFolderId || 'root');
    setBackupAccountID(task.accountId);
    setBackupSyncMode(task.syncMode);
    setModal({ type: 'backup-task' });
  };

  const handleRunSyncTaskNow = async (id: string, localPath: string) => {
    try {
      showToast(
        lang === 'id'
          ? `Memulai pemeriksaan & sinkronisasi backup untuk "${localPath}"...`
          : `Starting backup check & sync for "${localPath}"...`
      );
      // @ts-ignore
      await window.go.main.App.RunSyncTaskNowByID(id);
    } catch (err) {
      showInfoDialog("Error", "Failed to start backup sync: " + err);
    }
  };

  const handleRemoveSyncTask = (id: string, localPath: string) => {
    showConfirmDialog(
      lang === 'id' ? 'Hapus Tugas Backup' : 'Delete Backup Task',
      lang === 'id'
        ? `Apakah Anda yakin ingin menghapus tugas backup untuk folder "${localPath}"?`
        : `Are you sure you want to delete the backup task for "${localPath}"?`,
      async () => {
        try {
          // @ts-ignore
          await window.go.main.App.RemoveSyncTask(id);
          fetchSyncTasks();
          showToast(lang === 'id' ? 'Tugas backup berhasil dihapus' : 'Backup task removed');
        } catch (err) {
          showInfoDialog("Error", "Failed to remove backup task: " + err);
        }
      },
      {
        variant: 'danger',
        confirmLabel: lang === 'id' ? 'Hapus' : 'Delete',
        cancelLabel: lang === 'id' ? 'Batal' : 'Cancel',
      }
    );
  };

  const handleToggleSyncTask = async (id: string, enabled: boolean) => {
    try {
      // @ts-ignore
      await window.go.main.App.ToggleSyncTask(id, enabled);
      fetchSyncTasks();
    } catch (err) {
      showInfoDialog("Error", "Failed to toggle backup task: " + err);
    }
  };

  const fetchTrashedFiles = async (showFullLoading: boolean = true) => {
    if (showFullLoading) setLoading(true);
    try {
      const list = await GetTrashedFiles();
      setFiles(list || []);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const getOrderedAccounts = () => {
    const orderStr = settings['custom_account_order'] || '';
    if (!orderStr) return accounts;
    const orderArray = orderStr.split(',').map((id: string) => id.trim());
    
    // Sort accounts based on orderArray index
    const sorted = [...accounts].sort((a, b) => {
      let idxA = orderArray.indexOf(a.id);
      let idxB = orderArray.indexOf(b.id);
      if (idxA === -1) idxA = 999;
      if (idxB === -1) idxB = 999;
      return idxA - idxB;
    });
    return sorted;
  };

  const handleMoveAccount = async (index: number, direction: 'up' | 'down') => {
    const ordered = getOrderedAccounts();
    const newOrdered = [...ordered];
    const targetIndex = direction === 'up' ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= newOrdered.length) return;
    
    // Swap
    const temp = newOrdered[index];
    newOrdered[index] = newOrdered[targetIndex];
    newOrdered[targetIndex] = temp;
    
    // Save to settings
    const newOrderIDs = newOrdered.map(a => a.id).join(',');
    try {
      await SaveSetting("custom_account_order", newOrderIDs);
      await fetchSettings();
    } catch (err) {
      showInfoDialog("Error", "Failed to update account order: " + err);
    }
  };

  const toggleTheme = () => {
    const newTheme = theme === 'dark' ? 'light' : 'dark';
    setTheme(newTheme);
    document.body.setAttribute('data-theme', newTheme);
    localStorage.setItem('driverouter-theme', newTheme);
  };

  const fetchAccounts = async () => {
    try {
      const list = await GetAccounts();
      setAccounts(list || []);
    } catch (e) {
      console.error(e);
    }
  };

  const fetchSettings = async () => {
    try {
      const data = await GetSettings();
      setSettings(data || {});
      if (data) {
        if (data.language) {
          setLang(data.language as 'en' | 'id');
        }
        if (data.minimize_to_tray) {
          setMinToTray(data.minimize_to_tray === 'true');
        }
        if (data.auto_startup) {
          setAutoStartup(data.auto_startup === 'true');
        }
        if (data.backup_interval) {
          setBackupInterval(parseInt(data.backup_interval) || 60);
        }
      }
    } catch (e) {
      console.error(e);
    }
  };

  const fetchFiles = async (pId: string, showFullLoading: boolean = true) => {
    if (showFullLoading) setLoading(true);
    try {
      const list = await GetFiles(pId, false, searchKeyword);
      setFiles(list || []);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const fetchStarredFiles = async (showFullLoading: boolean = true) => {
    if (showFullLoading) setLoading(true);
    try {
      const list = await GetFiles('', true, searchKeyword);
      setFiles(list || []);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const handleSync = async () => {
    setSyncing(true);
    try {
      await SyncDrives();
      await fetchAccounts();
      if (view === 'explorer') fetchFiles(parentID);
      else if (view === 'starred') fetchStarredFiles();
      else if (view === 'shared') fetchFiles('__shared__');
      else if (view === 'trash') fetchTrashedFiles();
    } catch (e) {
      showInfoDialog("Error", t('syncError') + e);
    } finally {
      setSyncing(false);
    }
  };

  // Directory traversal
  const navigateToFolder = (folderId: string, folderName: string) => {
    setSelectedFile(null);
    setSearchKeyword('');
    setView('explorer');
    if (folderId === 'root') {
      setParentID('root');
      setBreadcrumbs([{ id: 'root', name: t('myDrive') }]);
    } else {
      const index = breadcrumbs.findIndex(b => b.id === folderId);
      if (index !== -1) {
        // Backwards breadcrumb click
        setParentID(folderId);
        setBreadcrumbs(breadcrumbs.slice(0, index + 1));
      } else {
        // Double click forward
        setParentID(folderId);
        setBreadcrumbs([...breadcrumbs, { id: folderId, name: folderName }]);
      }
    }
  };

  const handleToggleSelect = (id: string) => {
    setSelectedIDs(prev => {
      if (prev.includes(id)) {
        return prev.filter(x => x !== id);
      } else {
        return [...prev, id];
      }
    });
  };

  const handleToggleSelectAll = () => {
    if (selectedIDs.length === filteredFiles.length) {
      setSelectedIDs([]);
    } else {
      setSelectedIDs(filteredFiles.map(f => f.id));
    }
  };

  const handleBulkDelete = async () => {
    if (selectedIDs.length === 0) return;
    showConfirmDialog(
      "Confirm Delete",
      `Are you sure you want to delete ${selectedIDs.length} selected items? This will delete the files permanently from all physical cloud locations.`,
      async () => {
        try {
          for (const id of selectedIDs) {
            await DeleteFile(id);
          }
          setSelectedIDs([]);
          setSelectedFile(null);
          if (view === 'explorer') fetchFiles(parentID);
          else if (view === 'starred') fetchStarredFiles();
          else if (view === 'recent') fetchRecentFiles();
        } catch (err) {
          showInfoDialog("Error", "Failed to delete some files: " + err);
        }
      },
      { variant: 'danger', confirmLabel: t('moveToTrash') }
    );
  };

  const handleBulkDownload = async () => {
    if (selectedIDs.length === 0) return;
    try {
      await DownloadBulkDialog(selectedIDs);
      setSelectedIDs([]);
    } catch (err) {
      showInfoDialog("Error", "Bulk download error: " + err);
    }
  };

  const openTransferModal = async (file: FileRecord) => {
    setTransferFile(file);
    try {
      const folders = await GetVirtualFolders();
      setVirtualFolders(folders || []);
      const activeAccounts = accounts.filter(a => a.active);
      if (activeAccounts.length > 0) {
        const otherAcc = activeAccounts.find(a => a.id !== file.accountId);
        setSelectedDestAccountID(otherAcc ? otherAcc.id : activeAccounts[0].id);
      }
      setSelectedDestFolderID('root');
      setModal({ type: 'transfer-file' as any });
    } catch (err) {
      showInfoDialog("Error", "Failed to retrieve folder list: " + err);
    }
  };

  const handleTransferSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!transferFile || !selectedDestAccountID) return;
    setTransferLoading(true);
    try {
      await CopyFileToAccount(transferFile.id, selectedDestAccountID, selectedDestFolderID);
      setModal(null);
      setTransferFile(null);
      showInfoDialog("Success", `Direct transfer for '${transferFile.name}' has been scheduled in the background. Check the Transfers Progress drawer for details.`, 'info');
    } catch (err) {
      showInfoDialog("Error", "Failed to start direct transfer: " + err);
    } finally {
      setTransferLoading(false);
    }
  };

  const handleRemoteUploadSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!remoteUploadURL.trim() || !remoteUploadAccountID) return;
    setTransferLoading(true);
    try {
      await RemoteUploadFromURL(parentID, remoteUploadAccountID, remoteUploadURL);
      setModal(null);
      setRemoteUploadURL('');
      showInfoDialog("Success", "Remote URL upload has been scheduled in the background. Check the Transfers drawer for details.", 'info');
    } catch (err) {
      showInfoDialog("Error", "Failed to start remote upload: " + err);
    } finally {
      setTransferLoading(false);
    }
  };

  const handleCompressFilesSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!zipArchiveName.trim() || selectedIDs.length === 0) return;
    setTransferLoading(true);
    try {
      await CompressFilesToZip(selectedIDs, parentID, zipArchiveName);
      setModal(null);
      setZipArchiveName('');
      setSelectedIDs([]);
      showInfoDialog("Success", "Compression task has been scheduled in the background. Check the Transfers drawer for details.", 'info');
    } catch (err) {
      showInfoDialog("Error", "Failed to start compression: " + err);
    } finally {
      setTransferLoading(false);
    }
  };

  const handleExtractZip = async (file: FileRecord) => {
    try {
      await ExtractZipFile(file.id, parentID);
      showInfoDialog("Success", `Extraction of '${file.name}' has been scheduled in the background. Check the Transfers drawer for details.`, 'info');
    } catch (err) {
      showInfoDialog("Error", "Failed to start extraction: " + err);
    }
  };

  const scanDuplicateFiles = async () => {
    setDuplicatesLoading(true);
    try {
      const list = await FindDuplicateFiles();
      setDuplicateFiles(list || []);
    } catch (err) {
      showInfoDialog("Error", "Failed to scan for duplicate files: " + err);
    } finally {
      setDuplicatesLoading(false);
    }
  };

  const handleOpenInCloud = async (file: FileRecord) => {
    try {
      const url = await GetFileWebURL(file.id);
      if (url) {
        window.open(url, '_blank');
      } else {
        await OpenFileInBrowser(file.id);
      }
      fetchFiles(parentID);
    } catch (err) {
      showInfoDialog("Unsupported", "This provider does not support direct web links or the link could not be retrieved: " + err);
    }
  };

  const handleCopyShareLink = async (file: FileRecord) => {
    try {
      const url = await GetFileWebURL(file.id);
      await navigator.clipboard.writeText(url);
      showToast("Link copied to clipboard");
      fetchFiles(parentID);
    } catch (err) {
      showInfoDialog("Unsupported", "Could not retrieve share link for this item: " + err);
    }
  };

  const openShareModal = async (file: FileRecord) => {
    setShareFile(file);
    setShareEmail('');
    setShareRole('reader');
    setModal({ type: 'share' as any });
    setShareLoading(true);
    try {
      const perms = await GetFilePermissions(file.id);
      setSharePermissions(perms || []);
      
      const anyonePerm = perms.find((p: any) => p.type === 'anyone');
      if (anyonePerm) {
        setShareGeneralAccess('anyone');
        setShareGeneralRole(anyonePerm.role === 'writer' ? 'writer' : 'reader');
      } else {
        setShareGeneralAccess('restricted');
      }
    } catch (err) {
      console.error("Failed to load permissions:", err);
      showInfoDialog("Error", "Could not fetch permissions for this item: " + err);
      setModal(null);
    } finally {
      setShareLoading(false);
    }
  };

  const handleAddPermission = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!shareFile || !shareEmail.trim()) return;
    setShareLoading(true);
    try {
      await AddFilePermission(shareFile.id, shareEmail, shareRole);
      setShareEmail('');
      const perms = await GetFilePermissions(shareFile.id);
      setSharePermissions(perms || []);
      showToast("Access updated");
      fetchFiles(parentID);
    } catch (err) {
      showInfoDialog("Error", "Failed to add email: " + err);
    } finally {
      setShareLoading(false);
    }
  };

  const handleDeletePermission = async (permID: string) => {
    if (!shareFile) return;
    setShareLoading(true);
    try {
      await DeleteFilePermission(shareFile.id, permID);
      const perms = await GetFilePermissions(shareFile.id);
      setSharePermissions(perms || []);
      
      const anyonePerm = perms.find((p: any) => p.type === 'anyone');
      if (anyonePerm) {
        setShareGeneralAccess('anyone');
      } else {
        setShareGeneralAccess('restricted');
      }
      
      showToast("Permission removed");
      fetchFiles(parentID);
    } catch (err) {
      showInfoDialog("Error", "Failed to delete permission: " + err);
    } finally {
      setShareLoading(false);
    }
  };

  const handleGeneralAccessChange = async (newAccess: string, newRole?: 'reader' | 'writer') => {
    if (!shareFile) return;
    const resolvedRole = newRole || shareGeneralRole;
    setShareLoading(true);
    try {
      await SetFileGeneralAccess(shareFile.id, newAccess, resolvedRole);
      setShareGeneralAccess(newAccess);
      if (newRole) setShareGeneralRole(newRole);
      
      const perms = await GetFilePermissions(shareFile.id);
      setSharePermissions(perms || []);
      showToast("General access updated");
      fetchFiles(parentID);
    } catch (err) {
      showInfoDialog("Error", "Failed to change general access: " + err);
    } finally {
      setShareLoading(false);
    }
  };

  const fetchFileActivities = async (fileID: string) => {
    setActivitiesLoading(true);
    try {
      const list = await GetFileActivities(fileID);
      setFileActivities(list || []);
    } catch (err) {
      console.error(err);
    } finally {
      setActivitiesLoading(false);
    }
  };

  const fetchGeneralActivities = async () => {
    setGeneralActivitiesLoading(true);
    try {
      const list = await GetGeneralActivities(10);
      setGeneralActivities(list || []);
    } catch (err) {
      console.error(err);
    } finally {
      setGeneralActivitiesLoading(false);
    }
  };

  // Folder creation
  const handleCreateFolderSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!folderNameInput.trim()) return;

    try {
      await CreateFolder(parentID, folderNameInput);
      setFolderNameInput('');
      setModal(null);
      fetchFiles(parentID);
    } catch (err) {
      showInfoDialog("Error", "Error: " + err);
    }
  };

  // Upload dialog launcher
  const handleUploadFile = async (manualAccountId?: string) => {
    try {
      await UploadFileDialog(parentID, manualAccountId || "");
    } catch (err) {
      console.error("Upload error:", err);
    }
  };

  // Disconnect accounts
  const handleDisconnect = async (id: string) => {
    showConfirmDialog(
      t('confirm'),
      t('confirmDisconnect'),
      async () => {
        try {
          await DisconnectAccount(id);
          fetchAccounts();
          if (view === 'explorer') fetchFiles(parentID);
        } catch (err) {
          showInfoDialog("Error", t('disconnectError') + err);
        }
      },
      { variant: 'danger', confirmLabel: t('disconnect') }
    );
  };

  const handleToggleAccountActive = async (id: string) => {
    try {
      await ToggleAccountActive(id);
      fetchAccounts();
      if (view === 'explorer') fetchFiles(parentID);
    } catch (err: any) {
      showInfoDialog("Error", "Failed to toggle account status: " + (err?.message || String(err)));
    }
  };

  // Save Settings Strategy
  const handleStrategyChange = async (strategy: string) => {
    try {
      await SaveSetting("upload_strategy", strategy);
      fetchSettings();
    } catch (err) {
      showInfoDialog("Error", t('saveStrategyError') + err);
    }
  };

  // Save Settings Language
  const handleLanguageChange = async (newLang: 'en' | 'id') => {
    try {
      await SaveSetting("language", newLang);
      setLang(newLang);
      fetchSettings();
    } catch (err) {
      showInfoDialog("Error", t('saveLanguageError') + err);
    }
  };

  // Save Settings Minimize to Tray
  const handleMinimizeTrayChange = async (enable: boolean) => {
    try {
      await SaveSetting("minimize_to_tray", enable ? "true" : "false");
      setMinToTray(enable);
      fetchSettings();
    } catch (err) {
      showInfoDialog("Error", t('saveTrayError') + err);
    }
  };

  // Custom client credentials setup
  const openCredentialsModal = (provider: string) => {
    let cidKey = provider + '_client_id';
    let secretKey = provider + '_client_secret';
    if (provider === 'telegram_user') {
      cidKey = 'telegram_api_id';
      secretKey = 'telegram_api_hash';
    }
    const cid = settings[cidKey] || '';
    const secret = settings[secretKey] || '';
    setCredClientID(cid);
    setCredClientSecret(secret);
    setModal({ type: 'credentials', provider });
  };

  const handleCredentialsSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const provider = modal?.provider;
    if (!provider) return;

    try {
      await SaveCredentials(provider, credClientID, credClientSecret);
      await fetchSettings();
      setModal(null);
    } catch (err) {
      showInfoDialog("Error", "Failed to save credentials: " + err);
    }
  };

  const isConfigured = (provider: string) => {
    let cidKey = provider + '_client_id';
    let secretKey = provider + '_client_secret';
    if (provider === 'telegram_user') {
      cidKey = 'telegram_api_id';
      secretKey = 'telegram_api_hash';
    }
    return !!(settings[cidKey] && settings[secretKey]);
  };

  // Link accounts
  const handleLinkAccount = async (providerName: string) => {
    if (providerName === 'mega') {
      setMegaEmail('');
      setMegaPassword('');
      setMegaError('');
      setModal({ type: 'mega' });
      return;
    }
    if (providerName === 'koofr') {
      setKoofrUser('');
      setKoofrPass('');
      setKoofrError('');
      setModal({ type: 'koofr' });
      return;
    }
    if (providerName === 'mediafire') {
      setMediafireEmail('');
      setMediafirePassword('');
      setMediafireError('');
      setModal({ type: 'mediafire' });
      return;
    }
    if (providerName === 'fourshared') {
      setFoursharedEmail('');
      setFoursharedPassword('');
      setFoursharedError('');
      setModal({ type: 'fourshared' });
      return;
    }
    if (providerName === 'b2') {
      setB2DisplayName('');
      setB2KeyID('');
      setB2AppKey('');
      setB2Bucket('');
      setB2Error('');
      setModal({ type: 'b2' });
      return;
    }
    if (providerName === 'smb') {
      setSmbDisplayName('');
      setSmbHost('');
      setSmbShare('');
      setSmbUsername('');
      setSmbPassword('');
      setSmbError('');
      setModal({ type: 'smb' });
      return;
    }
    if (providerName === 'ftp') {
      setServerDisplayName('');
      setServerHost('');
      setServerPort(21);
      setServerUsername('');
      setServerPassword('');
      setServerBaseDir('/');
      setServerError('');
      setModal({ type: 'ftp' });
      return;
    }
    if (providerName === 'sftp') {
      setServerDisplayName('');
      setServerHost('');
      setServerPort(22);
      setServerUsername('');
      setServerPassword('');
      setServerBaseDir('/');
      setServerError('');
      setModal({ type: 'sftp' });
      return;
    }
    if (providerName === 'webdav') {
      setWebdavName('');
      setWebdavUrl('');
      setWebdavUsername('');
      setWebdavPassword('');
      setWebdavError('');
      setModal({ type: 'webdav' });
      return;
    }
    if (providerName === 's3') {
      setS3Name('');
      setS3Endpoint('');
      setS3Bucket('');
      setS3AccessKey('');
      setS3SecretKey('');
      setS3Error('');
      setModal({ type: 's3' });
      return;
    }
    if (providerName === 'telegram') {
      setTelegramName('');
      setTelegramToken('');
      setTelegramChatID('');
      setTelegramError('');
      setModal({ type: 'telegram' });
      return;
    }
    if (providerName === 'telegram_user') {
      setTgUserDisplayName('');
      setTgUserPhone('');
      setTgUserCode('');
      setTgUserPassword('');
      setTgUserStep('phone');
      setTgUserError('');
      setModal({ type: 'telegram_user' });
      return;
    }

    setModal(null);
    try {
      await StartOAuthFlow(providerName);
      fetchAccounts();
    } catch (err) {
      showInfoDialog("Error", "OAuth Error: " + err);
    }
  };

  // Connect WebDAV Submit
  const handleWebdavSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!webdavName.trim() || !webdavUrl.trim() || !webdavUsername.trim() || !webdavPassword.trim()) {
      return;
    }

    setWebdavLoading(true);
    setWebdavError('');
    try {
      await AddWebDAVAccount(webdavName, webdavUrl, webdavUsername, webdavPassword);
      setModal(null);
      fetchAccounts();
      if (view === 'explorer') fetchFiles(parentID);
    } catch (err: any) {
      setWebdavError(err?.message || String(err));
    } finally {
      setWebdavLoading(false);
    }
  };

  // Connect S3 Submit
  const handleS3Submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!s3Name.trim() || !s3Endpoint.trim() || !s3Bucket.trim() || !s3AccessKey.trim() || !s3SecretKey.trim()) {
      return;
    }

    setS3Loading(true);
    setS3Error('');
    try {
      await AddS3Account(s3Name, s3Endpoint, s3Bucket, s3AccessKey, s3SecretKey);
      setModal(null);
      fetchAccounts();
      if (view === 'explorer') fetchFiles(parentID);
    } catch (err: any) {
      setS3Error(err?.message || String(err));
    } finally {
      setS3Loading(false);
    }
  };

  // Connect Telegram Submit
  const handleTelegramSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!telegramName.trim() || !telegramToken.trim() || !telegramChatID.trim()) {
      return;
    }

    setTelegramLoading(true);
    setTelegramError('');
    try {
      await AddTelegramAccount(telegramName, telegramToken, telegramChatID);
      setModal(null);
      fetchAccounts();
      if (view === 'explorer') fetchFiles(parentID);
    } catch (err: any) {
      setTelegramError(err?.message || String(err));
    } finally {
      setTelegramLoading(false);
    }
  };

  // Telegram User Phone Submit
  const handleTelegramUserPhoneSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!tgUserPhone.trim() || !tgUserDisplayName.trim()) return;

    setTgUserLoading(true);
    setTgUserError('');
    try {
      await SendTelegramCode(tgUserPhone);
      setTgUserStep('code');
    } catch (err: any) {
      setTgUserError(err?.message || String(err));
    } finally {
      setTgUserLoading(false);
    }
  };

  // Telegram User Code Submit
  const handleTelegramUserCodeSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!tgUserCode.trim() && tgUserStep === 'code') return;

    setTgUserLoading(true);
    setTgUserError('');
    try {
      await VerifyTelegramCode(tgUserCode, tgUserPassword, tgUserDisplayName);
      setModal(null);
      fetchAccounts();
      if (view === 'explorer') fetchFiles(parentID);
    } catch (err: any) {
      const errMsg = err?.message || String(err);
      if (errMsg.includes("PASSWORD_REQUIRED")) {
        setTgUserStep('password');
      } else {
        setTgUserError(errMsg);
      }
    } finally {
      setTgUserLoading(false);
    }
  };

  // Context Menu handlers
  const handleContextMenu = (e: React.MouseEvent, file: FileRecord) => {
    e.preventDefault();
    e.stopPropagation();
    setSelectedFile(file);

    const menuWidth = 200;
    const menuHeight = 220;
    let x = e.clientX;
    let y = e.clientY;

    if (x + menuWidth > window.innerWidth) {
      x = window.innerWidth - menuWidth - 10;
    }
    if (y + menuHeight > window.innerHeight) {
      y = window.innerHeight - menuHeight - 10;
    }
    if (x < 0) x = 10;
    if (y < 0) y = 10;

    setContextMenu({
      visible: true,
      x: x,
      y: y,
      file: file
    });
  };

  const handleEmptySpaceContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();

    const menuWidth = 200;
    const menuHeight = 110;
    let x = e.clientX;
    let y = e.clientY;

    if (x + menuWidth > window.innerWidth) {
      x = window.innerWidth - menuWidth - 10;
    }
    if (y + menuHeight > window.innerHeight) {
      y = window.innerHeight - menuHeight - 10;
    }
    if (x < 0) x = 10;
    if (y < 0) y = 10;

    setContextMenu({
      visible: true,
      x: x,
      y: y,
      file: null
    });
  };

  const triggerFilePreview = async (file: FileRecord) => {
    setPreviewFile(file);
    setPreviewData(null);
    setPreviewLoading(true);
    setPreviewError('');
    try {
      const res = await PreviewFile(file.id);
      if (res && res.success) {
        setPreviewData(res);
      } else {
        const errMsg = res?.error || 'Failed to load preview';
        if (errMsg === 'UNSUPPORTED_TYPE') {
          setPreviewFile(null); // Close modal
          showConfirmDialog(
            t('previewError'),
            `${t('unsupportedPreview')} ${t('downloadConfirm')}`,
            () => {
              handleDownload(file);
            },
            {
              variant: 'warning',
              confirmLabel: t('downloadFile'),
              cancelLabel: t('cancel'),
            }
          );
          return;
        }
        setPreviewError(errMsg);
      }
    } catch (err: any) {
      const errMsg = err?.message || String(err);
      if (errMsg.includes('UNSUPPORTED_TYPE')) {
        setPreviewFile(null);
        showConfirmDialog(
          t('previewError'),
          `${t('unsupportedPreview')} ${t('downloadConfirm')}`,
          () => {
            handleDownload(file);
          },
          {
            variant: 'warning',
            confirmLabel: t('downloadFile'),
            cancelLabel: t('cancel'),
          }
        );
        return;
      }
      setPreviewError(errMsg);
    } finally {
      setPreviewLoading(false);
    }
  };

  const handleStarToggle = async (file: FileRecord) => {
    try {
      await ToggleStarred(file.id, !file.starred);
      if (view === 'explorer') fetchFiles(parentID);
      else if (view === 'starred') fetchStarredFiles();
      if (selectedFile?.id === file.id) {
        setSelectedFile({ ...file, starred: !file.starred });
      }
    } catch (err) {
      showInfoDialog("Error", "Star toggle error: " + err);
    }
  };

  const handleRename = async (file: FileRecord) => {
    showPromptDialog(
      t('renameItem'),
      "",
      t('name'),
      file.name,
      async (newName) => {
        if (newName && newName.trim() !== "" && newName !== file.name) {
          try {
            await RenameFile(file.id, newName);
            if (view === 'explorer') fetchFiles(parentID);
            else if (view === 'starred') fetchStarredFiles();
          } catch (err) {
            showInfoDialog("Error", t('renameError') + err);
          }
        }
      }
    );
  };

  const handleDelete = async (file: FileRecord) => {
    const isTrashSupported = ['google', 'onedrive', 'dropbox', 'box', 'yandex', 'pcloud'].includes(file.provider.toLowerCase());
    const confirmMsg = isTrashSupported
      ? t('confirmDelete').replace('{name}', file.name)
      : t('confirmPermanentDelete').replace('{name}', file.name);
    const confirmBtnLabel = isTrashSupported ? t('moveToTrash') : t('permanentlyDelete');

    showConfirmDialog(
      t('confirm'),
      confirmMsg,
      async () => {
        try {
          await DeleteFile(file.id);
          if (view === 'explorer') fetchFiles(parentID);
          else if (view === 'starred') fetchStarredFiles();
          if (selectedFile?.id === file.id) setSelectedFile(null);
        } catch (err) {
          showInfoDialog("Error", t('deleteError') + err);
        }
      },
      { variant: 'danger', confirmLabel: confirmBtnLabel }
    );
  };

  const handleRestoreFile = async (file: FileRecord) => {
    try {
      await RestoreFile(file.id);
      showToast(t('restore') + " success");
      fetchTrashedFiles();
      if (selectedFile?.id === file.id) setSelectedFile(null);
    } catch (e) {
      showInfoDialog("Error", "Failed to restore: " + e);
    }
  };

  const handlePermanentDelete = async (file: FileRecord) => {
    showConfirmDialog(
      t('permanentlyDelete'),
      t('confirmPermanentDelete').replace('{name}', file.name),
      async () => {
        try {
          await PermanentlyDeleteFile(file.id);
          showToast(t('permanentlyDelete') + " success");
          fetchTrashedFiles();
          if (selectedFile?.id === file.id) setSelectedFile(null);
        } catch (e) {
          showInfoDialog("Error", "Failed to permanently delete: " + e);
        }
      },
      { variant: 'danger', confirmLabel: t('permanentlyDelete') }
    );
  };

  const handleDownload = async (file: FileRecord) => {
    try {
      await DownloadFileDialog(file.id);
    } catch (err) {
      showInfoDialog("Error", t('downloadError') + err);
    }
  };

  // Helper storage size displays
  const formatBytes = (bytes: number, decimals = 2) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
  };

  const getAccountQuotaLabel = (acc: AccountRecord) => {
    if (acc.provider === 'telegram' || acc.provider === 'telegram_user') {
      return `${formatBytes(acc.usedSpace)} / ${t('quotaUnlimited')}`;
    }
    if (acc.totalSpace === 0 && acc.usedSpace === 0) {
      return t('quotaUnavailable');
    }
    if (acc.totalSpace === 0 || acc.totalSpace <= acc.usedSpace) {
      return `${formatBytes(acc.usedSpace)} ${t('quotaUsedOnly')}`;
    }

    return `${formatBytes(acc.usedSpace)} / ${formatBytes(acc.totalSpace)}`;
  };

  // Compute total aggregated capacity space
  const totalUsed = accounts.reduce((sum, a) => sum + (a.usedSpace || 0), 0);
  const totalLimit = accounts.reduce((sum, a) => sum + (a.totalSpace || 0), 0);
  const totalPercent = totalLimit > 0 ? (totalUsed / totalLimit) * 100 : 0;

  return (
    <div className="app-container">
      {/* Sidebar navigation */}
      <aside className={`sidebar ${sidebarCollapsed ? 'collapsed' : ''}`}>
        <div className="logo-section">
          <div className="logo-icon">
            <img src={logoImg} alt="Logo" style={{ width: '32px', height: '32px', objectFit: 'contain' }} />
          </div>
          {!sidebarCollapsed && <span className="logo-text">Awd DriveRouter</span>}
        </div>

        <div className="fab-container" ref={newDropdownRef}>
          <button className="fab-new" onClick={() => setShowNewDropdown(!showNewDropdown)} title={sidebarCollapsed ? t('new') : undefined}>
            <IconPlus />
            {!sidebarCollapsed && <span>{t('new')}</span>}
          </button>
          {showNewDropdown && (
            <div className="fab-dropdown">
              <div className="fab-dropdown-item" onClick={() => { setShowNewDropdown(false); setModal({ type: 'create-folder' }); }}>
                <IconFolder />
                <span>{t('newFolder')}</span>
              </div>
              <div className="fab-dropdown-item has-submenu">
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <IconDownload />
                  <span>{t('fileUpload')}</span>
                </div>
                <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor" style={{ marginLeft: '8px', color: 'var(--md-sys-color-on-surface-variant)' }}><path d="M8.59 16.59L13.17 12 8.59 7.41 10 6l6 6-6 6-1.41-1.41z"/></svg>
                <div className="submenu" style={{ minWidth: '220px', left: '100%', top: '0', marginLeft: '4px' }}>
                  <div className="context-item" onClick={() => { setShowNewDropdown(false); handleUploadFile(); }} style={{ padding: '8px 12px', fontSize: '13px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/></svg>
                    <span>{t('autoAllocate')}</span>
                  </div>
                  <div style={{ borderBottom: '1px solid var(--md-sys-color-outline-variant)', margin: '4px 0' }}></div>
                  {accounts.filter(a => a.active).map(acc => (
                    <div
                      key={acc.id}
                      className="context-item"
                      onClick={() => { setShowNewDropdown(false); handleUploadFile(acc.id); }}
                      style={{ padding: '8px 12px', fontSize: '13px', display: 'flex', alignItems: 'center', gap: '8px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
                    >
                      <div style={{ width: '16px', height: '16px', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                        {acc.provider === 'google' && <IconGoogleDrive />}
                        {acc.provider === 'onedrive' && <IconOneDrive />}
                        {acc.provider === 'dropbox' && <IconDropbox />}
                        {acc.provider === 'box' && <IconBox />}
                        {acc.provider === 'yandex' && <IconYandex />}
                        {acc.provider === 'pcloud' && <IconPCloud />}
                        {acc.provider === 'mega' && <IconMega />}
                        {(acc.provider === 'telegram' || acc.provider === 'telegram_user') && <IconTelegram />}
                        {!['google', 'onedrive', 'dropbox', 'box', 'yandex', 'pcloud', 'mega', 'telegram', 'telegram_user'].includes(acc.provider) && <IconCloud />}
                      </div>
                      <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {acc.displayName}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
              <div className="fab-dropdown-item" onClick={() => { setShowNewDropdown(false); const activeAcc = accounts.filter(a => a.active); if (activeAcc.length > 0) setRemoteUploadAccountID(activeAcc[0].id); setModal({ type: 'remote-upload' }); }}>
                <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" style={{ color: 'var(--md-sys-color-primary)' }}><path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM17 13l-5 5-5-5h3V9h4v4h3z"/></svg>
                <span>Remote URL Upload</span>
              </div>
            </div>
          )}
        </div>

        <nav className="sidebar-nav">
          <div className={`nav-item ${view === 'home' ? 'active' : ''}`} onClick={() => setView('home')} title={sidebarCollapsed ? t('home') : undefined}>
            <IconHome />
            {!sidebarCollapsed && <span>{t('home')}</span>}
          </div>
          <div className={`nav-item ${view === 'explorer' ? 'active' : ''}`} onClick={() => navigateToFolder('root', t('myDrive'))} title={sidebarCollapsed ? t('myDrive') : undefined}>
            <IconFolder />
            {!sidebarCollapsed && <span>{t('myDrive')}</span>}
          </div>
          <div className={`nav-item ${view === 'shared' ? 'active' : ''}`} onClick={() => setView('shared')} title={sidebarCollapsed ? t('shared') : undefined}>
            <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M15 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm-9-2c1.66 0 3-1.34 3-3S7.66 5 6 5 3 6.34 3 8s1.34 3 3 3zm0 4.2c-2.33 0-7 1.17-7 3.5V20h14v-2.3c0-2.33-4.67-3.5-7-3.5zm9 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V20h6v-2.3c0-2.33-4.67-3.5-7-3.5z"/></svg>
            {!sidebarCollapsed && <span>{t('shared')}</span>}
          </div>
          <div className={`nav-item ${view === 'recent' ? 'active' : ''}`} onClick={() => setView('recent')} title={sidebarCollapsed ? t('recent') : undefined}>
            <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67z"/></svg>
            {!sidebarCollapsed && <span>{t('recent')}</span>}
          </div>
          <div className={`nav-item ${view === 'starred' ? 'active' : ''}`} onClick={() => setView('starred')} title={sidebarCollapsed ? t('starred') : undefined}>
            <IconStar />
            {!sidebarCollapsed && <span>{t('starred')}</span>}
          </div>
          <div className={`nav-item ${view === 'webshare' ? 'active' : ''}`} onClick={() => setView('webshare')} title={sidebarCollapsed ? t('webSharing') : undefined}>
            <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/></svg>
            {!sidebarCollapsed && <span>{t('webSharing')}</span>}
          </div>
          <div className={`nav-item ${view === 'trash' ? 'active' : ''}`} onClick={() => setView('trash')} title={sidebarCollapsed ? t('trash') : undefined}>
            <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M16 9v10H8V9h8m-1.5-6h-5l-1 1H5v2h14V4h-3.5l-1-1zM18 7H6v12c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7z"/></svg>
            {!sidebarCollapsed && <span>{t('trash')}</span>}
          </div>
          <div className={`nav-item ${view === 'settings' ? 'active' : ''}`} onClick={() => setView('settings')} title={sidebarCollapsed ? t('settings') : undefined}>
            <IconSettings />
            {!sidebarCollapsed && <span>{t('settings')}</span>}
          </div>
          <div className={`nav-item ${view === 'about' ? 'active' : ''}`} onClick={() => setView('about')} title={sidebarCollapsed ? t('about') : undefined}>
            <IconInfo />
            {!sidebarCollapsed && <span>{t('about')}</span>}
          </div>
        </nav>

        {/* Quota widget at bottom */}
        {sidebarCollapsed ? (
          <div
            className={`quota-widget-collapsed ${view === 'storage' ? 'active' : ''}`}
            onClick={() => setView('storage')}
            title={t('storage')}
          >
            <IconCloud />
          </div>
        ) : (
          <div className={`quota-widget ${view === 'storage' ? 'active' : ''}`} onClick={() => setView('storage')}>
            <div className="quota-title">
              <IconCloud />
              <span>{t('storageUsage')}</span>
            </div>
            <div className="quota-bar-bg">
              <div className="quota-bar-fill" style={{ width: `${totalPercent}%` }}></div>
            </div>
            <div className="quota-text">
              {formatBytes(totalUsed)} of {formatBytes(totalLimit)} used
            </div>
          </div>
        )}
      </aside>

      {/* Main Panel content */}
      <main className="main-panel">
        <div className="top-toolbar">
          <div className="toolbar-left">
            <button className="icon-btn hamburger-btn" onClick={() => setSidebarCollapsed(!sidebarCollapsed)} title="Toggle sidebar">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                <path d="M3 18h18v-2H3v2zm0-5h18v-2H3v2zm0-7v2h18V6H3z"/>
              </svg>
            </button>
          </div>
          
          <div className="search-container">
            <div className="search-icon"><IconSearch /></div>
            <input
              type="text"
              className="search-input"
              placeholder={t('searchPlaceholder')}
              value={localSearchTerm}
              onChange={(e) => {
                setLocalSearchTerm(e.target.value);
                if (view !== 'explorer' && view !== 'starred') {
                  setView('explorer');
                }
              }}
            />
            {localSearchTerm && (
              <button
                className="search-clear-btn"
                onClick={() => {
                  setLocalSearchTerm('');
                  setSearchKeyword('');
                }}
                title="Clear search"
              >
                <IconClose />
              </button>
            )}

            {/* Filter Toggle Button */}
            <button
              type="button"
              className={`search-filter-btn ${advancedSearchOpen ? 'active' : ''}`}
              onClick={() => setAdvancedSearchOpen(!advancedSearchOpen)}
              title="Advanced search filters"
              style={{
                position: 'absolute',
                right: '44px',
                top: '50%',
                transform: 'translateY(-50%)',
                background: 'none',
                border: 'none',
                color: (filterAccountID !== 'all' || filterFileType !== 'all' || filterFileSize !== 'all' || filterModDate !== 'all') ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-on-surface-variant)',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                padding: '6px',
                borderRadius: '50%',
                transition: 'background-color var(--transition-fast)'
              }}
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                <path d="M3 17v2h6v-2H3zM3 5v2h10V5H3zm10 16v-2h8v-2h-8v-2h-2v6h2zM7 9v2H3v2h4v2h2V9H7zm14 4v-2H11v2h10zm-6-4h2V7h4V5h-4V3h-2v6z"/>
              </svg>
            </button>

            {/* Advanced Search Dropdown Panel */}
            {advancedSearchOpen && (
              <div 
                className="search-filter-panel" 
                style={{
                  position: 'absolute',
                  top: 'calc(100% + 8px)',
                  left: 0,
                  right: 0,
                  backgroundColor: 'var(--md-sys-color-surface-container-high)',
                  border: '1px solid var(--md-sys-color-outline-variant)',
                  borderRadius: '16px',
                  boxShadow: 'var(--shadow-3)',
                  padding: '20px',
                  zIndex: 200,
                  display: 'flex',
                  flexDirection: 'column',
                  gap: '16px'
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid var(--md-sys-color-outline-variant)', paddingBottom: '8px' }}>
                  <span style={{ fontWeight: '600', fontSize: '14px', color: 'var(--md-sys-color-on-surface)' }}>Filter Search Results</span>
                  {(filterAccountID !== 'all' || filterFileType !== 'all' || filterFileSize !== 'all' || filterModDate !== 'all') && (
                    <button
                      type="button"
                      onClick={() => {
                        setFilterAccountID('all');
                        setFilterFileType('all');
                        setFilterFileSize('all');
                        setFilterModDate('all');
                      }}
                      style={{
                        background: 'none',
                        border: 'none',
                        color: 'var(--md-sys-color-error)',
                        fontSize: '12px',
                        fontWeight: '500',
                        cursor: 'pointer',
                        padding: '4px 8px',
                        borderRadius: '4px'
                      }}
                    >
                      Clear Filters
                    </button>
                  )}
                </div>

                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
                  {/* Account Filter */}
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                    <label style={{ fontSize: '11px', fontWeight: '600', color: 'var(--md-sys-color-primary)', letterSpacing: '0.5px' }}>CLOUD ACCOUNT</label>
                    <select
                      value={filterAccountID}
                      onChange={(e) => setFilterAccountID(e.target.value)}
                      style={{
                        height: '38px',
                        width: '100%',
                        boxSizing: 'border-box',
                        padding: '0 12px',
                        borderRadius: '8px',
                        border: '1px solid var(--md-sys-color-outline-variant)',
                        backgroundColor: 'var(--md-sys-color-surface)',
                        color: 'var(--md-sys-color-on-surface)',
                        fontSize: '13px',
                        outline: 'none',
                        cursor: 'pointer'
                      }}
                    >
                      <option value="all">All Accounts</option>
                      {accounts.filter(a => a.active).map(acc => (
                        <option key={acc.id} value={acc.id}>{acc.displayName} ({acc.provider.toUpperCase()})</option>
                      ))}
                    </select>
                  </div>

                  {/* Type Filter */}
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                    <label style={{ fontSize: '11px', fontWeight: '600', color: 'var(--md-sys-color-primary)', letterSpacing: '0.5px' }}>FILE TYPE</label>
                    <select
                      value={filterFileType}
                      onChange={(e) => setFilterFileType(e.target.value)}
                      style={{
                        height: '38px',
                        width: '100%',
                        boxSizing: 'border-box',
                        padding: '0 12px',
                        borderRadius: '8px',
                        border: '1px solid var(--md-sys-color-outline-variant)',
                        backgroundColor: 'var(--md-sys-color-surface)',
                        color: 'var(--md-sys-color-on-surface)',
                        fontSize: '13px',
                        outline: 'none',
                        cursor: 'pointer'
                      }}
                    >
                      <option value="all">All Types</option>
                      <option value="image">Images (jpg, png, etc.)</option>
                      <option value="video">Videos (mp4, mkv, etc.)</option>
                      <option value="audio">Audios (mp3, wav, etc.)</option>
                      <option value="document">Documents (pdf, docx, txt, etc.)</option>
                      <option value="other">Others</option>
                    </select>
                  </div>

                  {/* Size Filter */}
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                    <label style={{ fontSize: '11px', fontWeight: '600', color: 'var(--md-sys-color-primary)', letterSpacing: '0.5px' }}>FILE SIZE</label>
                    <select
                      value={filterFileSize}
                      onChange={(e) => setFilterFileSize(e.target.value)}
                      style={{
                        height: '38px',
                        width: '100%',
                        boxSizing: 'border-box',
                        padding: '0 12px',
                        borderRadius: '8px',
                        border: '1px solid var(--md-sys-color-outline-variant)',
                        backgroundColor: 'var(--md-sys-color-surface)',
                        color: 'var(--md-sys-color-on-surface)',
                        fontSize: '13px',
                        outline: 'none',
                        cursor: 'pointer'
                      }}
                    >
                      <option value="all">Any Size</option>
                      <option value="gt10mb">&gt; 10 MB</option>
                      <option value="gt100mb">&gt; 100 MB</option>
                      <option value="gt1gb">&gt; 1 GB</option>
                    </select>
                  </div>

                  {/* Date Filter */}
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                    <label style={{ fontSize: '11px', fontWeight: '600', color: 'var(--md-sys-color-primary)', letterSpacing: '0.5px' }}>MODIFIED DATE</label>
                    <select
                      value={filterModDate}
                      onChange={(e) => setFilterModDate(e.target.value)}
                      style={{
                        height: '38px',
                        width: '100%',
                        boxSizing: 'border-box',
                        padding: '0 12px',
                        borderRadius: '8px',
                        border: '1px solid var(--md-sys-color-outline-variant)',
                        backgroundColor: 'var(--md-sys-color-surface)',
                        color: 'var(--md-sys-color-on-surface)',
                        fontSize: '13px',
                        outline: 'none',
                        cursor: 'pointer'
                      }}
                    >
                      <option value="all">Any Time</option>
                      <option value="today">Today (Last 24 hrs)</option>
                      <option value="yesterday">Yesterday</option>
                      <option value="week">Last 7 Days</option>
                      <option value="month">Last 30 Days</option>
                    </select>
                  </div>
                </div>

                <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', borderTop: '1px solid var(--md-sys-color-outline-variant)', paddingTop: '12px', marginTop: '4px' }}>
                  <button
                    type="button"
                    onClick={() => setAdvancedSearchOpen(false)}
                    style={{
                      padding: '8px 16px',
                      borderRadius: '8px',
                      border: 'none',
                      backgroundColor: 'var(--md-sys-color-primary)',
                      color: 'var(--md-sys-color-on-primary)',
                      fontSize: '13px',
                      fontWeight: '600',
                      cursor: 'pointer'
                    }}
                  >
                    Apply Filters
                  </button>
                </div>
              </div>
            )}
          </div>

          <div className="toolbar-actions">
            <button className={`icon-btn ${syncing ? 'spinning' : ''}`} onClick={handleSync} title={t('syncTooltip')}>
              <IconRefresh />
            </button>
            <button className="icon-btn theme-toggle-btn" onClick={toggleTheme} title={t('switchTheme')}>
              {theme === 'dark' ? <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor"><path d="M12 7c-2.76 0-5 2.24-5 5s2.24 5 5 5 5-2.24 5-5-2.24-5-5-5zM2 13h2c.55 0 1-.45 1-1s-.45-1-1-1H2c-.55 0-1 .45-1 1s.45 1 1 1zm18 0h2c.55 0 1-.45 1-1s-.45-1-1-1h-2c-.55 0-1 .45-1 1s.45 1 1 1zM11 2v2c0 .55.45 1 1 1s1-.45 1-1V2c0-.55-.45-1-1-1s-1 .45-1 1zm0 18v2c0 .55.45 1 1 1s1-.45 1-1v-2c0-.55-.45-1-1-1s-1 .45-1 1zM5.99 4.58c-.39-.39-1.03-.39-1.41 0s-.39 1.03 0 1.41l1.06 1.06c.39.39 1.03.37 1.41-.02.39-.39.39-1.03 0-1.41L5.99 4.58zm12.37 12.37c-.39-.39-1.03-.39-1.41 0s-.39 1.03 0 1.41l1.06 1.06c.39.39 1.03.37 1.41-.02.39-.39.39-1.03 0-1.41l-1.06-1.06zm1.06-12.37c-.39-.39-1.03-.39-1.41 0l-1.06 1.06c-.39.39-.39 1.03 0 1.41.39.39 1.03.39 1.41 0l1.06-1.06c.39-.38.39-1.02 0-1.41zM5.99 16.95l-1.06 1.06c-.39.39-.39 1.02 0 1.41.39.39 1.03.39 1.41 0l1.06-1.06c.39-.39.39-1.03 0-1.41a.996.996 0 0 0-1.41 0z"/></svg> : <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor"><path d="M12.3 22c5.07 0 9.2-4.13 9.2-9.2 0-3.07-1.51-5.79-3.83-7.48-.46-.34-1.11.13-.93.68.83 2.56.24 5.4-1.63 7.27-1.88 1.88-4.71 2.46-7.27 1.63-.55-.18-1.02.47-.68.93C9.57 20.49 12.3 22 12.3 22z"/></svg>}
            </button>
            <button className="icon-btn" onClick={() => setView('settings')} title={t('settings')}>
              <IconSettings />
            </button>
          </div>
        </div>

        {/* Dashboard/Home Panel */}
        {view === 'home' && (
          <div className="explorer-body">
            <div className="explorer-header">
              <h2 style={{ fontSize: '24px', fontWeight: '500' }}>{t('dashboardOverview')}</h2>
            </div>
            
            <div className="content-scroll">
              <div className="dashboard-grid">
                <div className="dashboard-card">
                  <h3 className="section-title">{t('quotasAllocation')}</h3>
                  <div className="provider-list">
                    {accounts.length === 0 ? (
                      <p style={{ fontSize: '13px', color: 'var(--md-sys-color-on-surface-variant)' }}>{t('noAccounts')}</p>
                    ) : (
                      accounts.map(acc => {
                        const percent = acc.totalSpace > 0 ? (acc.usedSpace / acc.totalSpace) * 100 : 0;
                        return (
                          <div key={acc.id} className="provider-row">
                            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                              {acc.provider === 'google' && <IconGoogleDrive />}
                              {acc.provider === 'onedrive' && <IconOneDrive />}
                              {acc.provider === 'dropbox' && <IconDropbox />}
                              {acc.provider === 'box' && <IconBox />}
                              {acc.provider === 'yandex' && <IconYandex />}
                              {acc.provider === 'pcloud' && <IconPCloud />}
                              {acc.provider === 'mega' && <IconMega />}
                              {acc.provider === 'telegram' && <IconTelegram />}
                              {acc.provider === 'telegram_user' && <IconTelegram />}
                              {acc.provider === 's3' && <IconSettings />}
                              {acc.provider === 'webdav' && <IconCloud />}
                              <div>
                                <h4 style={{ fontSize: '14px', fontWeight: '600' }}>{acc.displayName}</h4>
                                {acc.email && (
                                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                                    <span style={{ fontSize: '11px', color: 'var(--md-sys-color-on-surface-variant)' }}>
                                      {showEmails[acc.id] ? acc.email : maskEmail(acc.email)}
                                    </span>
                                    <button
                                      type="button"
                                      onClick={() => toggleShowEmail(acc.id)}
                                      title={showEmails[acc.id] ? (t('hideEmail') || "Sembunyikan Email") : (t('showEmail') || "Tampilkan Email")}
                                      style={{ background: 'none', border: 'none', cursor: 'pointer', padding: 0, color: 'var(--md-sys-color-on-surface-variant)', opacity: 0.7, display: 'inline-flex', alignItems: 'center' }}
                                    >
                                      {showEmails[acc.id] ? <IconEyeOff /> : <IconEye />}
                                    </button>
                                  </div>
                                )}
                              </div>
                            </div>
                            <div style={{ width: '150px', textAlign: 'right' }}>
                              <div className="quota-bar-bg" style={{ height: '6px', margin: '4px 0' }}>
                                <div className="quota-bar-fill" style={{ width: `${percent}%` }}></div>
                              </div>
                              <span style={{ fontSize: '11px' }}>{getAccountQuotaLabel(acc)}</span>
                            </div>
                          </div>
                        );
                      })
                    )}
                  </div>
                </div>

                <div className="dashboard-card">
                  <h3 className="section-title">{t('uploadRouterStatus')}</h3>
                  <p className="dashboard-desc">
                    {t('uploadsRoutedRule')}: <strong>{settings.upload_strategy?.replace('_', ' ').toUpperCase() || 'ROUND ROBIN'}</strong>. 
                  </p>
                  <div style={{ padding: '16px', borderRadius: '12px', border: '1px solid var(--md-sys-color-outline-variant)', backgroundColor: 'var(--md-sys-color-surface)' }}>
                    <h4 style={{ fontSize: '14px', fontWeight: '600', marginBottom: '8px' }}>{t('activeProviders')}:</h4>
                    {accounts.filter(a => a.active).length === 0 ? (
                      <div className="alert-panel">
                        <span>{t('noActiveProviders')}</span>
                      </div>
                    ) : (
                      <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
                        {accounts.filter(a => a.active).map(a => (
                          <span key={a.id} className={`provider-badge badge-${a.provider}`}>{a.provider}</span>
                        ))}
                      </div>
                    )}
                  </div>
                </div>

                <div className="dashboard-card" style={{ gridColumn: 'span 2' }}>
                  <h3 className="section-title">{t('recentActivity')}</h3>
                  {generalActivitiesLoading && (
                    <p style={{ fontSize: '13px', color: 'var(--md-sys-color-on-surface-variant)' }}>Loading...</p>
                  )}
                  {!generalActivitiesLoading && generalActivities.length === 0 && (
                    <p style={{ fontSize: '13px', color: 'var(--md-sys-color-on-surface-variant)' }}>{t('noRecentActivity')}</p>
                  )}
                  {!generalActivitiesLoading && generalActivities.length > 0 && (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', maxHeight: '200px', overflowY: 'auto' }}>
                      {generalActivities.map(act => (
                        <div key={act.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '10px 14px', borderRadius: '8px', backgroundColor: 'var(--md-sys-color-surface-container-high)', fontSize: '12px' }}>
                          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', minWidth: 0 }}>
                            <span style={{
                              fontWeight: '600',
                              textTransform: 'uppercase',
                              fontSize: '8px',
                              padding: '2px 6px',
                              borderRadius: '4px',
                              backgroundColor: act.action === 'share' ? 'var(--md-sys-color-primary-container)' : act.action === 'rename' ? 'var(--md-sys-color-secondary-container)' : 'var(--md-sys-color-surface-container-highest)',
                              color: act.action === 'share' ? 'var(--md-sys-color-on-primary-container)' : act.action === 'rename' ? 'var(--md-sys-color-on-secondary-container)' : 'var(--md-sys-color-on-surface)',
                              flexShrink: 0
                            }}>
                              {act.action}
                            </span>
                            <span style={{ color: 'var(--md-sys-color-on-surface)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={translateActivityDetails(act.details)}>
                              {translateActivityDetails(act.details)}
                            </span>
                          </div>
                          <span style={{ fontSize: '10px', color: 'var(--md-sys-color-on-surface-variant)', flexShrink: 0, marginLeft: '8px' }}>
                            {formatDateTime(act.timestamp)}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>

              {/* Quick Actions / Recent Folders */}
              <div style={{ marginTop: '32px' }}>
                <h3 className="section-title">{t('quickNavigation')}</h3>
                <div className="folders-grid">
                  <div className="folder-card" onClick={() => navigateToFolder('root', t('myDrive'))}>
                    <div className="folder-icon"><IconFolder /></div>
                    <span className="folder-name">{t('mainStorageRoot')}</span>
                  </div>
                  <div className="folder-card" onClick={() => setView('starred')}>
                    <div className="folder-icon" style={{ color: '#f4b400' }}><IconStar filled /></div>
                    <span className="folder-name">{t('starredDocuments')}</span>
                  </div>
                  <div className="folder-card" onClick={() => setView('storage')}>
                    <div className="folder-icon"><IconCloud /></div>
                    <span className="folder-name">{t('storage')}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* File Explorer (My Drive / Starred / Shared / Recent views) */}
        {(view === 'explorer' || view === 'starred' || view === 'shared' || view === 'recent' || view === 'trash') && (
          <div className="explorer-body" style={{ position: 'relative' }}>
            {loading && files.length > 0 && (
              <div className="top-linear-progress">
                <div className="top-linear-progress-bar"></div>
              </div>
            )}
            <div className="explorer-header">
              {view === 'explorer' && (
                <div className="breadcrumbs">
                  {breadcrumbs.map((crumb, idx) => (
                    <React.Fragment key={crumb.id}>
                      {idx > 0 && <span className="breadcrumb-separator"><IconChevronRight /></span>}
                      <span
                        className={`breadcrumb-item ${idx === breadcrumbs.length - 1 ? 'breadcrumb-active' : ''}`}
                        onClick={() => navigateToFolder(crumb.id, crumb.name)}
                      >
                        {crumb.name}
                      </span>
                    </React.Fragment>
                  ))}
                </div>
              )}
              {view === 'starred' && (
                <h2 style={{ fontSize: '20px', fontWeight: '500' }}>{t('starred')}</h2>
              )}
              {view === 'shared' && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                  <h2 style={{ fontSize: '20px', fontWeight: '500', margin: 0 }}>{t('shared')}</h2>
                  <p style={{ margin: 0, fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)' }}>{t('sharedSupportNote')}</p>
                  <p style={{ margin: 0, fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)' }}>{t('sharedSupportedProviders')}</p>
                </div>
              )}
              {view === 'recent' && (
                <h2 style={{ fontSize: '20px', fontWeight: '500' }}>{t('recent')}</h2>
              )}
              {view === 'trash' && (
                <h2 style={{ fontSize: '20px', fontWeight: '500' }}>{t('trash')}</h2>
              )}

              <div className="explorer-tools" style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                {selectedIDs.length > 0 && (
                  <div style={{ display: 'flex', gap: '8px' }}>
                    <button
                      className="btn btn-text"
                      style={{ color: 'var(--md-sys-color-primary)', display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', padding: '6px 12px', border: '1px solid var(--md-sys-color-primary)', borderRadius: '100px', cursor: 'pointer' }}
                      onClick={handleBulkDownload}
                      title="Download Selected"
                    >
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM17 13l-5 5-5-5h3V9h4v4h3z"/></svg>
                      <span>Download ({selectedIDs.length})</span>
                    </button>
                    <button
                      className="btn btn-text"
                      style={{ color: 'var(--md-sys-color-error)', display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', padding: '6px 12px', border: '1px solid var(--md-sys-color-error)', borderRadius: '100px', cursor: 'pointer' }}
                      onClick={handleBulkDelete}
                      title="Delete Selected"
                    >
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/></svg>
                      <span>Delete ({selectedIDs.length})</span>
                    </button>
                  </div>
                )}
                <button 
                  className="icon-btn" 
                  onClick={() => setActiveLayout(activeLayout === 'list' ? 'grid' : 'list')} 
                  title={activeLayout === 'list' ? "Grid View" : "List View"}
                >
                  {activeLayout === 'list' ? (
                    <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor"><path d="M4 11h5V5H4v6zm0 8h5v-6H4v6zm7 0h5v-6h-5v6zm8 0h5v-6h-5v6zm-8-8h5V5h-5v6zm8 0h5V5h-5v6z"/></svg>
                  ) : (
                    <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor"><path d="M3 15h18v-2H3v2zm0 4h18v-2H3v2zm0-8h18V9H3v2zm0-6v2h18V5H3z"/></svg>
                  )}
                </button>
                {selectedFile && (
                  <button className="icon-btn" onClick={() => setDetailsSidebar(!detailsSidebar)} title="Toggle Details">
                    <IconInfo />
                  </button>
                )}
              </div>
            </div>

            {/* Content & Details sidebar wrapper */}
            <div className="explorer-content-wrapper">
              {/* Browser area */}
              <div className="content-scroll" onContextMenu={handleEmptySpaceContextMenu}>
              {view === 'recent' && (
                <div className="recent-filters-bar" style={{ display: 'flex', gap: '8px', marginBottom: '20px', flexWrap: 'wrap', padding: '0 4px' }}>
                  <button
                    type="button"
                    className={`chip-btn ${recentAccountFilter === 'all' ? 'active' : ''}`}
                    onClick={() => setRecentAccountFilter('all')}
                    style={{
                      padding: '6px 16px',
                      borderRadius: '100px',
                      fontSize: '13px',
                      fontWeight: '500',
                      cursor: 'pointer',
                      border: '1px solid var(--md-sys-color-outline-variant)',
                      background: recentAccountFilter === 'all' ? 'var(--md-sys-color-primary-container)' : 'var(--md-sys-color-surface-container-low)',
                      color: recentAccountFilter === 'all' ? 'var(--md-sys-color-on-primary-container)' : 'var(--md-sys-color-on-surface)',
                      transition: 'all var(--transition-fast)'
                    }}
                  >
                    All Providers
                  </button>
                  {accounts.filter(a => a.active).map(acc => (
                    <button
                      type="button"
                      key={acc.id}
                      className={`chip-btn ${recentAccountFilter === acc.id ? 'active' : ''}`}
                      onClick={() => setRecentAccountFilter(acc.id)}
                      style={{
                        padding: '6px 16px',
                        borderRadius: '100px',
                        fontSize: '13px',
                        fontWeight: '500',
                        cursor: 'pointer',
                        border: '1px solid var(--md-sys-color-outline-variant)',
                        background: recentAccountFilter === acc.id ? 'var(--md-sys-color-primary-container)' : 'var(--md-sys-color-surface-container-low)',
                        color: recentAccountFilter === acc.id ? 'var(--md-sys-color-on-primary-container)' : 'var(--md-sys-color-on-surface)',
                        transition: 'all var(--transition-fast)'
                      }}
                    >
                      {acc.displayName} ({acc.provider.toUpperCase()})
                    </button>
                  ))}
                </div>
              )}

              {loading && files.length === 0 ? (
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', marginTop: '100px', gap: '16px' }}>
                  <div className="loading-spinner" style={{ width: '40px', height: '40px', border: '3px solid var(--md-sys-color-surface-variant)', borderTop: '3px solid var(--md-sys-color-primary)', borderRadius: '50%', animation: 'spin 1s linear infinite' }}></div>
                  <p style={{ fontSize: '14px', color: 'var(--md-sys-color-on-surface-variant)' }}>{t('loadingData')}</p>
                  <style>{`
                    @keyframes spin {
                      0% { transform: rotate(0deg); }
                      100% { transform: rotate(360deg); }
                    }
                  `}</style>
                </div>
              ) : filteredFiles.length === 0 ? (
                <div style={{ textAlign: 'center', marginTop: '80px', color: 'var(--md-sys-color-on-surface-variant)' }}>
                  <IconFolder />
                  <p style={{ marginTop: '12px', fontSize: '15px' }}>
                    {view === 'shared' ? 'No shared files found' : view === 'recent' ? 'No recent files found' : view === 'trash' ? t('noTrashedFiles') : t('folderEmpty')}
                  </p>
                  {view === 'shared' && (
                    <p style={{ fontSize: '12px', maxWidth: '520px', margin: '0 auto' }}>
                      {t('sharedSupportNote')} {t('sharedSupportedProviders')}
                    </p>
                  )}
                  {view !== 'shared' && view !== 'recent' && view !== 'trash' && (
                    <p style={{ fontSize: '12px' }}>{t('folderEmptyDesc')}</p>
                  )}
                </div>
              ) : view === 'recent' ? (
                /* Option 3 Chronological recent files grouped */
                <div style={{ display: 'flex', flexDirection: 'column', gap: '32px' }}>
                  {groupRecentFiles(getFilteredRecentFiles()).map(group => (
                    <div key={group.label} className="recent-group">
                      <h3 style={{ fontSize: '14px', fontWeight: '600', color: 'var(--md-sys-color-primary)', borderBottom: '1px solid var(--md-sys-color-outline-variant)', paddingBottom: '8px', marginBottom: '16px', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
                        {group.label}
                      </h3>
                      {activeLayout === 'grid' ? (
                        <div className="files-grid">
                          {group.items.map(f => (
                            f.isFolder ? (
                              <div
                                key={f.id}
                                className={`folder-card ${selectedFile?.id === f.id ? 'selected' : ''}`}
                                onClick={() => setSelectedFile(f)}
                                onDoubleClick={() => navigateToFolder(f.id, f.name)}
                                onContextMenu={(e) => handleContextMenu(e, f)}
                                style={{ position: 'relative' }}
                              >
                                <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexGrow: 1, minWidth: 0 }}>
                                  <div className="file-card-checkbox" onClick={(e) => e.stopPropagation()} style={{ position: 'static', opacity: selectedIDs.includes(f.id) ? 1 : undefined }}>
                                    <input
                                      type="checkbox"
                                      checked={selectedIDs.includes(f.id)}
                                      onChange={() => handleToggleSelect(f.id)}
                                      style={{ width: '15px', height: '15px', cursor: 'pointer' }}
                                    />
                                  </div>
                                  <div className="folder-icon" style={{ flexShrink: 0 }}><FolderIconWithShared shared={f.shared} /></div>
                                  <span className="folder-name" title={f.name} style={{ flexGrow: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{f.name}</span>
                                </div>
                              </div>
                            ) : (
                              <div
                                key={f.id}
                                className={`file-card ${selectedFile?.id === f.id ? 'selected' : ''}`}
                                onClick={() => setSelectedFile(f)}
                                onDoubleClick={() => triggerFilePreview(f)}
                                onContextMenu={(e) => handleContextMenu(e, f)}
                              >
                                <div className="file-card-checkbox" onClick={(e) => e.stopPropagation()} style={{ opacity: selectedIDs.includes(f.id) ? 1 : undefined }}>
                                  <input
                                    type="checkbox"
                                    checked={selectedIDs.includes(f.id)}
                                    onChange={() => handleToggleSelect(f.id)}
                                    style={{ width: '15px', height: '15px', cursor: 'pointer' }}
                                  />
                                </div>
                                <div className="file-preview-placeholder">
                                  <FileIcon name={f.name} />
                                </div>
                                <div className="file-card-info">
                                  <span className="file-card-name" title={f.name}>{f.name}</span>
                                  <span className="file-card-provider" style={{ fontSize: '11px', color: 'var(--md-sys-color-on-surface-variant)' }}>
                                    {accounts.find(a => a.id === f.accountId)?.displayName || f.provider}
                                  </span>
                                </div>
                              </div>
                            )
                          ))}
                        </div>
                      ) : (
                        <table className="files-table">
                          <thead>
                            <tr>
                              <th style={{ width: '40px', paddingLeft: '12px' }}>
                                <input
                                  type="checkbox"
                                  checked={group.items.every(item => selectedIDs.includes(item.id))}
                                  onChange={() => {
                                    const groupIDs = group.items.map(item => item.id);
                                    const allSelected = groupIDs.every(id => selectedIDs.includes(id));
                                    if (allSelected) {
                                      setSelectedIDs(prev => prev.filter(id => !groupIDs.includes(id)));
                                    } else {
                                      setSelectedIDs(prev => [...new Set([...prev, ...groupIDs])]);
                                    }
                                  }}
                                  style={{ cursor: 'pointer' }}
                                />
                              </th>
                              <th>{t('name')}</th>
                              <th>{t('cloudProvider')}</th>
                              <th>{t('size')}</th>
                              <th>{t('lastModified')}</th>
                              <th style={{ width: '40px' }}></th>
                            </tr>
                          </thead>
                          <tbody>
                            {group.items.map(f => (
                              <tr
                                key={f.id}
                                className={`file-row ${selectedFile?.id === f.id ? 'selected' : ''}`}
                                onClick={() => setSelectedFile(f)}
                                onDoubleClick={() => f.isFolder ? navigateToFolder(f.id, f.name) : triggerFilePreview(f)}
                                onContextMenu={(e) => handleContextMenu(e, f)}
                              >
                                <td onClick={(e) => e.stopPropagation()} style={{ paddingLeft: '12px' }}>
                                  <input
                                    type="checkbox"
                                    checked={selectedIDs.includes(f.id)}
                                    onChange={() => handleToggleSelect(f.id)}
                                    style={{ cursor: 'pointer' }}
                                  />
                                </td>
                                <td>
                                  <div className="file-name-cell">
                                    <span style={{ display: 'flex', color: 'var(--md-sys-color-on-surface-variant)' }}>
                                      {f.isFolder ? <FolderIconWithShared shared={f.shared} /> : <FileIcon name={f.name} />}
                                    </span>
                                    <span style={{ wordBreak: 'break-all' }}>{f.name}</span>
                                    {f.starred && <span style={{ display: 'flex', color: '#f4b400' }}><IconStar filled /></span>}
                                  </div>
                                </td>
                                <td>
                                  {(() => {
                                    const acc = accounts.find(a => a.id === f.accountId);
                                    return (
                                      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: 'var(--md-sys-color-on-surface)' }}>
                                        {acc ? acc.displayName : f.provider}
                                      </div>
                                    );
                                  })()}
                                </td>
                                <td>{f.isFolder ? '—' : formatBytes(f.size)}</td>
                                <td>{formatDateTime(f.modifiedAt)}</td>
                                <td>
                                  <button className="icon-btn" onClick={(e) => { e.stopPropagation(); handleContextMenu(e, f); }}>
                                    <IconDots />
                                  </button>
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      )}
                    </div>
                  ))}
                </div>
              ) : activeLayout === 'grid' ? (
                <div>
                  {/* Folders block */}
                  {filteredFiles.filter(f => f.isFolder).length > 0 && (
                    <div className="folders-section">
                      <h3 className="section-title">{t('folders')}</h3>
                      <div className="folders-grid">
                        {filteredFiles.filter(f => f.isFolder).map(f => (
                          <div
                            key={f.id}
                            className={`folder-card ${selectedFile?.id === f.id ? 'selected' : ''}`}
                            onClick={() => setSelectedFile(f)}
                            onDoubleClick={() => navigateToFolder(f.id, f.name)}
                            onContextMenu={(e) => handleContextMenu(e, f)}
                            style={{ position: 'relative' }}
                          >
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexGrow: 1, minWidth: 0 }}>
                              <div className="file-card-checkbox" onClick={(e) => e.stopPropagation()} style={{ position: 'static', opacity: selectedIDs.includes(f.id) ? 1 : undefined }}>
                                <input
                                  type="checkbox"
                                  checked={selectedIDs.includes(f.id)}
                                  onChange={() => handleToggleSelect(f.id)}
                                  style={{ width: '15px', height: '15px', cursor: 'pointer' }}
                                />
                              </div>
                              <div className="folder-icon" style={{ flexShrink: 0 }}><FolderIconWithShared shared={f.shared} /></div>
                              <span className="folder-name" title={f.name} style={{ flexGrow: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{f.name}</span>
                            </div>
                            <button 
                              className="icon-btn folder-dots-btn" 
                              style={{ width: '24px', height: '24px', flexShrink: 0, opacity: 0, transition: 'opacity var(--transition-fast)' }}
                              onClick={(e) => { e.stopPropagation(); handleContextMenu(e, f); }}
                            >
                              <IconDots />
                            </button>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Files block */}
                  {filteredFiles.filter(f => !f.isFolder).length > 0 && (
                    <div>
                      <h3 className="section-title">Files</h3>
                      <div className="files-grid">
                        {filteredFiles.filter(f => !f.isFolder).map(f => (
                          <div
                            key={f.id}
                            className={`file-card ${selectedFile?.id === f.id ? 'selected' : ''}`}
                            onClick={() => setSelectedFile(f)}
                            onDoubleClick={() => triggerFilePreview(f)}
                            onContextMenu={(e) => handleContextMenu(e, f)}
                          >
                            {/* Checkbox */}
                            <div className="file-card-checkbox" onClick={(e) => e.stopPropagation()}>
                              <input
                                type="checkbox"
                                checked={selectedIDs.includes(f.id)}
                                onChange={() => handleToggleSelect(f.id)}
                                style={{ width: '16px', height: '16px', cursor: 'pointer' }}
                              />
                            </div>

                            {/* Provider Badge */}
                            <div className="file-card-provider">
                              <span className={`provider-badge badge-${f.provider}`} style={{ fontSize: '9px', padding: '2px 6px' }}>
                                {f.provider}
                              </span>
                            </div>

                            {/* Preview Area */}
                            <div className="file-card-preview">
                              <FileIcon name={f.name} />
                            </div>

                            {/* Info Area */}
                            <div className="file-card-info">
                              <div className="file-card-header">
                                <span className="file-card-name" title={f.name}>{f.name}</span>
                                <button 
                                  className="icon-btn" 
                                  style={{ width: '24px', height: '24px', flexShrink: 0 }}
                                  onClick={(e) => { e.stopPropagation(); handleContextMenu(e, f); }}
                                >
                                  <IconDots />
                                </button>
                              </div>
                              <div className="file-card-meta">
                                <span>{formatBytes(f.size)}</span>
                                <span>{new Date(f.modifiedAt).toLocaleDateString()}</span>
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              ) : (
                /* List Layout table */
                <table className="files-table">
                  <thead>
                    <tr>
                      <th style={{ width: '40px', paddingLeft: '12px' }}>
                        <input
                          type="checkbox"
                          checked={selectedIDs.length === filteredFiles.length && filteredFiles.length > 0}
                          onChange={handleToggleSelectAll}
                          style={{ cursor: 'pointer' }}
                        />
                      </th>
                      <th>{t('name')}</th>
                      <th>{t('cloudProvider')}</th>
                      <th>{t('size')}</th>
                      <th>{t('lastModified')}</th>
                      <th style={{ width: '40px' }}></th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredFiles.map(f => (
                      <tr
                        key={f.id}
                        className={`file-row ${selectedFile?.id === f.id ? 'selected' : ''}`}
                        onClick={() => setSelectedFile(f)}
                        onDoubleClick={() => f.isFolder ? navigateToFolder(f.id, f.name) : triggerFilePreview(f)}
                        onContextMenu={(e) => handleContextMenu(e, f)}
                      >
                        <td onClick={(e) => e.stopPropagation()} style={{ paddingLeft: '12px' }}>
                          <input
                            type="checkbox"
                            checked={selectedIDs.includes(f.id)}
                            onChange={() => handleToggleSelect(f.id)}
                            style={{ cursor: 'pointer' }}
                          />
                        </td>
                        <td>
                          <div className="file-name-cell">
                            <span style={{ display: 'flex', color: 'var(--md-sys-color-on-surface-variant)' }}>
                              {f.isFolder ? <FolderIconWithShared shared={f.shared} /> : <FileIcon name={f.name} />}
                            </span>
                            <span style={{ wordBreak: 'break-all' }}>{f.name}</span>
                            {f.starred && <span style={{ display: 'flex', color: '#f4b400' }}><IconStar filled /></span>}
                          </div>
                        </td>
                        <td>
                          {(() => {
                            const acc = accounts.find(a => a.id === f.accountId);
                            return (
                              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: 'var(--md-sys-color-on-surface)' }}>
                                <div style={{ display: 'flex', width: '24px', height: '24px', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                                  {f.provider === 'google' && <IconGoogleDrive />}
                                  {f.provider === 'onedrive' && <IconOneDrive />}
                                  {f.provider === 'dropbox' && <IconDropbox />}
                                  {f.provider === 'box' && <IconBox />}
                                  {f.provider === 'yandex' && <IconYandex />}
                                  {f.provider === 'pcloud' && <IconPCloud />}
                                  {f.provider === 'mega' && <IconMega />}
                                  {f.provider === 'koofr' && <IconKoofr />}
                                  {f.provider === 'mediafire' && <IconMediaFire />}
                                  {f.provider === 'fourshared' && <Icon4Shared />}
                                  {f.provider === 'b2' && <IconB2 />}
                                  {f.provider === 'smb' && <IconSmb />}
                                  {f.provider === 'ftp' && <IconFtp />}
                                  {f.provider === 'sftp' && <IconSftp />}
                                  {(f.provider === 'telegram' || f.provider === 'telegram_user') && <IconTelegram />}
                                  {f.provider === 'virtual' && <IconFolder />}
                                  {!['google', 'onedrive', 'dropbox', 'box', 'yandex', 'pcloud', 'mega', 'koofr', 'mediafire', 'fourshared', 'b2', 'smb', 'ftp', 'sftp', 'telegram', 'telegram_user', 'virtual'].includes(f.provider) && <IconCloud />}
                                </div>
                                <span 
                                  style={{ fontSize: '13px', fontWeight: '500', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: '160px' }} 
                                  title={acc ? `${acc.displayName} (${acc.email})` : f.provider}
                                >
                                  {acc ? acc.displayName : f.provider}
                                </span>
                              </div>
                            );
                          })()}
                        </td>
                        <td>{f.isFolder ? '—' : formatBytes(f.size)}</td>
                        <td>{formatDateTime(f.modifiedAt)}</td>
                        <td>
                          <button className="icon-btn" onClick={(e) => { e.stopPropagation(); handleContextMenu(e, f); }}>
                            <IconDots />
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>

            {/* Sidebar file details overlay */}
            {detailsSidebar && selectedFile && (
              <div className="details-sidebar" style={{ display: 'flex', flexDirection: 'column' }}>
                <div className="details-header" style={{ flexShrink: 0 }}>
                  <span className="details-title">{t('details')}</span>
                  <button className="icon-btn" onClick={() => setDetailsSidebar(false)}><IconClose /></button>
                </div>
                
                {/* Tab switcher */}
                <div style={{ display: 'flex', borderBottom: '1px solid var(--md-sys-color-outline-variant)', marginBottom: '16px', gap: '8px', flexShrink: 0 }}>
                  <div
                    onClick={() => setDetailsTab('details')}
                    style={{
                      flex: 1,
                      padding: '8px 0',
                      textAlign: 'center',
                      cursor: 'pointer',
                      fontSize: '13px',
                      fontWeight: '600',
                      color: detailsTab === 'details' ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-on-surface-variant)',
                      borderBottom: detailsTab === 'details' ? '2px solid var(--md-sys-color-primary)' : '2px solid transparent',
                      transition: 'all 0.2s'
                    }}
                  >
                    {t('details')}
                  </div>
                  <div
                    onClick={() => setDetailsTab('activity')}
                    style={{
                      flex: 1,
                      padding: '8px 0',
                      textAlign: 'center',
                      cursor: 'pointer',
                      fontSize: '13px',
                      fontWeight: '600',
                      color: detailsTab === 'activity' ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-on-surface-variant)',
                      borderBottom: detailsTab === 'activity' ? '2px solid var(--md-sys-color-primary)' : '2px solid transparent',
                      transition: 'all 0.2s'
                    }}
                  >
                    {t('activityTab')}
                  </div>
                </div>

                <div style={{ flexGrow: 1, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: '12px' }}>
                  {detailsTab === 'details' ? (
                    <>
                      <div className="details-file-icon" style={{ display: 'flex', justifyContent: 'center', padding: '16px 0', flexShrink: 0 }}>
                        {selectedFile.isFolder ? <FolderIconWithShared shared={selectedFile.shared} /> : <FileIcon name={selectedFile.name} />}
                      </div>
                      <div className="details-info-row">
                        <span className="details-label">{t('name')}</span>
                        <span className="details-value" style={{ wordBreak: 'break-all' }}>{selectedFile.name}</span>
                      </div>
                      <div className="details-info-row">
                        <span className="details-label">{t('type')}</span>
                        <span className="details-value">{selectedFile.isFolder ? t('newFolder') : 'File'}</span>
                      </div>
                      {!selectedFile.isFolder && (
                        <div className="details-info-row">
                          <span className="details-label">{t('size')}</span>
                          <span className="details-value">{formatBytes(selectedFile.size)}</span>
                        </div>
                      )}
                      <div className="details-info-row">
                        <span className="details-label">{t('cloudProvider')}</span>
                        <span className="details-value" style={{ textTransform: 'capitalize' }}>{selectedFile.provider}</span>
                      </div>
                      {['google', 'onedrive', 'dropbox', 'box', 'yandex'].includes(selectedFile.provider.toLowerCase()) && (
                        <div className="details-info-row">
                          <span className="details-label">{t('starred')}</span>
                          <span className="details-value">{selectedFile.starred ? t('yes') : t('no')}</span>
                        </div>
                      )}
                      <div className="details-info-row">
                        <span className="details-label">{t('createdAt')}</span>
                        <span className="details-value">{formatDateTime(selectedFile.createdAt)}</span>
                      </div>
                      <div className="details-info-row">
                        <span className="details-label">{t('modifiedAt')}</span>
                        <span className="details-value">{formatDateTime(selectedFile.modifiedAt)}</span>
                      </div>
                    </>
                  ) : (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', paddingRight: '4px' }}>
                      {activitiesLoading && (
                        <div style={{ textAlign: 'center', color: 'var(--md-sys-color-on-surface-variant)', fontSize: '13px', padding: '20px 0' }}>
                          {t('loadingActivities')}
                        </div>
                      )}
                      {!activitiesLoading && fileActivities.length === 0 && (
                        <div style={{ textAlign: 'center', color: 'var(--md-sys-color-on-surface-variant)', fontSize: '13px', padding: '20px 0' }}>
                          {t('noActivityHistory')}
                        </div>
                      )}
                      {!activitiesLoading && fileActivities.map(act => (
                        <div key={act.id} style={{ display: 'flex', gap: '8px', padding: '8px 12px', borderRadius: '8px', backgroundColor: 'var(--md-sys-color-surface-container)', fontSize: '12px', lineHeight: '1.4' }}>
                          <div style={{ display: 'flex', flexDirection: 'column', flexGrow: 1 }}>
                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '4px' }}>
                              <span style={{
                                fontWeight: '600',
                                textTransform: 'uppercase',
                                fontSize: '8px',
                                padding: '2px 6px',
                                borderRadius: '4px',
                                backgroundColor: act.action === 'share' ? 'var(--md-sys-color-primary-container)' : act.action === 'rename' ? 'var(--md-sys-color-secondary-container)' : 'var(--md-sys-color-surface-container-highest)',
                                color: act.action === 'share' ? 'var(--md-sys-color-on-primary-container)' : act.action === 'rename' ? 'var(--md-sys-color-on-secondary-container)' : 'var(--md-sys-color-on-surface)'
                              }}>
                                {act.action}
                              </span>
                              <span style={{ fontSize: '10px', color: 'var(--md-sys-color-on-surface-variant)' }}>
                                {new Date(act.timestamp).toLocaleDateString(lang === 'id' ? 'id-ID' : 'en-US')}
                              </span>
                            </div>
                            <p style={{ margin: 0, color: 'var(--md-sys-color-on-surface)' }}>
                              {translateActivityDetails(act.details)}
                            </p>
                            <span style={{ fontSize: '9px', color: 'var(--md-sys-color-on-surface-variant)', marginTop: '4px', alignSelf: 'flex-end' }}>
                              {new Date(act.timestamp).toLocaleTimeString(lang === 'id' ? 'id-ID' : 'en-US', { hour12: false })}
                            </span>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            )}
            </div>
          </div>
        )}

        {/* Storage tab (consolidated Cloud integrations + strategies) */}
        {view === 'storage' && (
          <div className="explorer-body">
            <div className="explorer-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '16px' }}>
              <div>
                <h2 style={{ fontSize: '22px', fontWeight: '500' }}>{t('storage')}</h2>
                <p style={{ fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)', marginTop: '4px' }}>
                  View all cloud accounts, their quotas, then connect or disconnect accounts.
                </p>
              </div>
              <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
                <button className={`btn btn-text ${syncing ? 'spinning' : ''}`} onClick={handleSync} style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px' }}>
                  <IconRefresh />
                  <span>{t('syncNow')}</span>
                </button>
                <button className="btn btn-filled" onClick={() => setModal({ type: 'add-account' })} style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                  <IconPlus />
                  <span>{t('connect')}</span>
                </button>
              </div>
            </div>

            {/* Sub-tabs buttons */}
            <div style={{ display: 'flex', gap: '12px', marginTop: '16px', borderBottom: '1px solid var(--md-sys-color-outline-variant)', paddingBottom: '12px' }}>
              <button
                className="btn"
                style={{
                  borderRadius: '100px',
                  padding: '8px 20px',
                  fontSize: '13px',
                  fontWeight: '500',
                  border: storageTab === 'overview' ? 'none' : '1px solid var(--md-sys-color-outline-variant)',
                  backgroundColor: storageTab === 'overview' ? 'var(--md-sys-color-primary-container)' : 'transparent',
                  color: storageTab === 'overview' ? 'var(--md-sys-color-on-primary-container)' : 'var(--md-sys-color-on-surface-variant)',
                  cursor: 'pointer'
                }}
                onClick={() => setStorageTab('overview')}
              >
                {t('overview')}
              </button>
              <button
                className="btn"
                style={{
                  borderRadius: '100px',
                  padding: '8px 20px',
                  fontSize: '13px',
                  fontWeight: '500',
                  border: storageTab === 'allocation' ? 'none' : '1px solid var(--md-sys-color-outline-variant)',
                  backgroundColor: storageTab === 'allocation' ? 'var(--md-sys-color-primary-container)' : 'transparent',
                  color: storageTab === 'allocation' ? 'var(--md-sys-color-on-primary-container)' : 'var(--md-sys-color-on-surface-variant)',
                  cursor: 'pointer'
                }}
                onClick={() => setStorageTab('allocation')}
              >
                {t('allocation')}
              </button>
              <button
                className="btn"
                style={{
                  borderRadius: '100px',
                  padding: '8px 20px',
                  fontSize: '13px',
                  fontWeight: '500',
                  border: storageTab === 'duplicates' ? 'none' : '1px solid var(--md-sys-color-outline-variant)',
                  backgroundColor: storageTab === 'duplicates' ? 'var(--md-sys-color-primary-container)' : 'transparent',
                  color: storageTab === 'duplicates' ? 'var(--md-sys-color-on-primary-container)' : 'var(--md-sys-color-on-surface-variant)',
                  cursor: 'pointer'
                }}
                onClick={() => setStorageTab('duplicates')}
              >
                Duplicates finder
              </button>
            </div>
            
            <div className="content-scroll" style={{ marginTop: '20px' }}>
              {storageTab === 'overview' && (
                <div>
                  {/* Aggregated Total Storage Bar */}
                  <div style={{ padding: '20px', borderRadius: '16px', backgroundColor: 'var(--md-sys-color-surface-container-low)', border: '1px solid var(--md-sys-color-outline-variant)', marginBottom: '24px' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
                      <span style={{ fontSize: '14px', fontWeight: '500', display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <IconCloud />
                        Total storage: {formatBytes(totalUsed)} / {formatBytes(totalLimit)}
                      </span>
                      <div style={{ fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)' }}>
                        <span style={{ marginRight: '16px' }}>Used space: <strong>{formatBytes(totalUsed)}</strong></span>
                        <span>Free space: <strong>{formatBytes(totalLimit - totalUsed)}</strong></span>
                      </div>
                    </div>
                    <div className="quota-bar-bg" style={{ height: '12px', borderRadius: '6px' }}>
                      <div className="quota-bar-fill" style={{ width: `${totalPercent}%`, borderRadius: '6px' }}></div>
                    </div>
                  </div>

                  {/* Connected Accounts Grid */}
                  {accounts.length === 0 ? (
                    <div style={{ textAlign: 'center', marginTop: '60px', color: 'var(--md-sys-color-on-surface-variant)' }}>
                      <IconCloud />
                      <p style={{ marginTop: '12px', fontSize: '15px' }}>{t('noAccounts')}</p>
                    </div>
                  ) : (
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))', gap: '20px' }}>
                      {accounts.map(acc => {
                        const percent = acc.totalSpace > 0 ? (acc.usedSpace / acc.totalSpace) * 100 : 0;
                        const freeSpace = acc.totalSpace - acc.usedSpace;
                        return (
                          <div key={acc.id} style={{ display: 'flex', flexDirection: 'column', padding: '20px', borderRadius: '16px', border: '1px solid var(--md-sys-color-outline-variant)', backgroundColor: 'var(--md-sys-color-surface-container)', justifyContent: 'space-between' }}>
                            <div>
                              {/* Header: Provider Icon & Title */}
                              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '16px' }}>
                                <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
                                  <div style={{ padding: '8px', borderRadius: '8px', backgroundColor: 'var(--md-sys-color-surface-container-high)' }}>
                                    {acc.provider === 'google' && <IconGoogleDrive />}
                                    {acc.provider === 'onedrive' && <IconOneDrive />}
                                    {acc.provider === 'dropbox' && <IconDropbox />}
                                    {acc.provider === 'box' && <IconBox />}
                                    {acc.provider === 'yandex' && <IconYandex />}
                                    {acc.provider === 'pcloud' && <IconPCloud />}
                                    {acc.provider === 'mega' && <IconMega />}
                                    {acc.provider === 'koofr' && <IconKoofr />}
                                    {acc.provider === 'mediafire' && <IconMediaFire />}
                                    {acc.provider === 'fourshared' && <Icon4Shared />}
                                    {acc.provider === 'b2' && <IconB2 />}
                                    {acc.provider === 'smb' && <IconSmb />}
                                    {acc.provider === 'ftp' && <IconFtp />}
                                    {acc.provider === 'sftp' && <IconSftp />}
                                    {acc.provider === 'telegram' && <IconTelegram />}
                                    {acc.provider === 'telegram_user' && <IconTelegram />}
                                    {acc.provider === 's3' && <IconSettings />}
                                    {acc.provider === 'webdav' && <IconCloud />}
                                  </div>
                                  <div>
                                    <h4 style={{ fontSize: '14px', fontWeight: '600', margin: 0 }}>{acc.displayName}</h4>
                                    <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                                      <span style={{ fontSize: '11px', color: 'var(--md-sys-color-on-surface-variant)', textTransform: 'capitalize' }}>
                                        {acc.provider.replace('_', ' ')}
                                      </span>
                                      {acc.email && (
                                        <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                                          <span style={{ fontSize: '11px', color: 'var(--md-sys-color-primary)', fontWeight: '500' }}>
                                            {showEmails[acc.id] ? acc.email : maskEmail(acc.email)}
                                          </span>
                                          <button
                                            type="button"
                                            onClick={() => toggleShowEmail(acc.id)}
                                            title={showEmails[acc.id] ? (t('hideEmail') || "Sembunyikan Email") : (t('showEmail') || "Tampilkan Email")}
                                            style={{ background: 'none', border: 'none', cursor: 'pointer', padding: '2px', color: 'var(--md-sys-color-on-surface-variant)', opacity: 0.75, display: 'inline-flex', alignItems: 'center' }}
                                          >
                                            {showEmails[acc.id] ? <IconEyeOff /> : <IconEye />}
                                          </button>
                                        </div>
                                      )}
                                    </div>
                                  </div>
                                </div>
                                <button
                                   type="button"
                                   onClick={() => handleToggleAccountActive(acc.id)}
                                   title={acc.active ? t('clickToDeactivate') : t('clickToActivate')}
                                   style={{ display: 'inline-flex', alignItems: 'center', gap: '6px', fontSize: '11px', fontWeight: '500', padding: '4px 10px', borderRadius: '100px', backgroundColor: acc.active ? 'rgba(52, 168, 83, 0.15)' : 'rgba(128, 128, 128, 0.15)', color: acc.active ? '#34a853' : '#808080', border: 'none', cursor: 'pointer', transition: 'all 0.2s' }}
                                 >
                                   <span style={{ width: '6px', height: '6px', borderRadius: '50%', backgroundColor: acc.active ? '#34a853' : '#808080' }}></span>
                                   {acc.active ? t('activeStatus') : t('inactiveStatus')}
                                 </button>
                              </div>

                              {/* Progress bar and Space details */}
                              <div style={{ marginBottom: '16px' }}>
                                <div className="quota-bar-bg" style={{ height: '6px', marginBottom: '8px' }}>
                                  <div className="quota-bar-fill" style={{ width: `${percent}%` }}></div>
                                </div>
                                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '11px', color: 'var(--md-sys-color-on-surface-variant)' }}>
                                  <span>{getAccountQuotaLabel(acc)}</span>
                                  <span style={{ fontWeight: '500', color: 'var(--md-sys-color-primary)' }}>
                                    {acc.provider === 'telegram' || acc.provider === 'telegram_user'
                                      ? t('quotaUnlimited')
                                      : acc.totalSpace > 0 && acc.totalSpace > acc.usedSpace
                                      ? `${formatBytes(freeSpace)} free`
                                      : t('unlimitedQuotaLabel')}
                                  </span>
                                </div>
                              </div>
                            </div>

                            {/* Actions block */}
                            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', borderTop: '1px solid var(--md-sys-color-outline-variant)', paddingTop: '12px' }}>
                              {!acc.active && ['google', 'onedrive', 'dropbox', 'box', 'yandex', 'pcloud'].includes(acc.provider) && (
                                <button className="btn btn-text" style={{ fontSize: '12px', color: 'var(--md-sys-color-primary)', fontWeight: '600' }} onClick={() => handleLinkAccount(acc.provider)}>
                                  {t('reauthenticate') || "Sambungkan Ulang"}
                                </button>
                              )}
                              {acc.provider !== 'webdav' && acc.provider !== 's3' && acc.provider !== 'telegram' && acc.provider !== 'telegram_user' && (
                                <button className="btn btn-text" style={{ fontSize: '12px' }} onClick={() => openCredentialsModal(acc.provider)}>{t('apiKeys')}</button>
                              )}
                              <button className="btn btn-text" style={{ color: 'var(--md-sys-color-error)', fontSize: '12px' }} onClick={() => handleDisconnect(acc.id)}>{t('disconnect')}</button>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>
              )}

              {storageTab === 'allocation' && (
                <div>
                  <h3 style={{ fontSize: '16px', fontWeight: '600', marginBottom: '8px' }}>{t('allocationRule')}</h3>
                  <p style={{ fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)', marginBottom: '20px' }}>
                    {t('allocationRuleDesc')}
                  </p>

                  {/* Strategies Grid */}
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: '16px', marginBottom: '32px' }}>
                    {[
                      { val: 'round_robin', label: 'Round Robin', desc: 'Uploads take turns across accounts in order.' },
                      { val: 'weighted_round_robin', label: 'Weighted Round Robin', desc: 'Larger accounts receive proportionally more uploads.' },
                      { val: 'least_used', label: 'Least Used', desc: 'Sends each upload to the account with lowest used %.' },
                      { val: 'max_free', label: 'Most Free Space', desc: 'Sends each upload to the account with most free space.' },
                      { val: 'custom_order', label: 'Custom Order', desc: 'Fills accounts in the exact order you set below.' }
                    ].map(st => {
                      const isActive = (settings.upload_strategy || 'round_robin') === st.val;
                      return (
                        <div
                          key={st.val}
                          onClick={() => handleStrategyChange(st.val)}
                          style={{
                            padding: '16px',
                            borderRadius: '12px',
                            border: isActive ? '2px solid var(--md-sys-color-primary)' : '1px solid var(--md-sys-color-outline-variant)',
                            backgroundColor: isActive ? 'var(--md-sys-color-primary-container)' : 'var(--md-sys-color-surface-container)',
                            color: isActive ? 'var(--md-sys-color-on-primary-container)' : 'var(--md-sys-color-on-surface)',
                            cursor: 'pointer',
                            transition: 'all 0.2s',
                            position: 'relative'
                          }}
                        >
                          <h4 style={{ fontSize: '14px', fontWeight: '600', marginBottom: '6px' }}>{st.label}</h4>
                          <p style={{ fontSize: '11px', lineHeight: '1.4', opacity: 0.85 }}>{st.desc}</p>
                          {isActive && (
                            <span style={{ position: 'absolute', top: '8px', right: '8px', color: 'var(--md-sys-color-primary)', fontSize: '14px' }}>✓</span>
                          )}
                        </div>
                      );
                    })}
                  </div>

                  {/* Custom Order Priortization Section */}
                  {(settings.upload_strategy === 'custom_order') && (
                    <div style={{ padding: '24px', borderRadius: '16px', border: '1px solid var(--md-sys-color-outline-variant)', backgroundColor: 'var(--md-sys-color-surface-container-low)' }}>
                      <h4 style={{ fontSize: '15px', fontWeight: '600', marginBottom: '4px' }}>{t('customOrderHeader')}</h4>
                      <p style={{ fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)', marginBottom: '20px' }}>
                        {t('customOrderDesc2')}
                      </p>

                      <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                        {getOrderedAccounts().map((acc, index, arr) => {
                          const freeSpace = acc.totalSpace - acc.usedSpace;
                          return (
                            <div key={acc.id} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 16px', borderRadius: '10px', border: '1px solid var(--md-sys-color-outline-variant)', backgroundColor: 'var(--md-sys-color-surface)' }}>
                              <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                                <span style={{ fontSize: '13px', fontWeight: '600', color: 'var(--md-sys-color-on-surface-variant)', width: '20px' }}>
                                  {index + 1}
                                </span>
                                <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                                  {acc.provider === 'google' && <IconGoogleDrive />}
                                  {acc.provider === 'onedrive' && <IconOneDrive />}
                                  {acc.provider === 'dropbox' && <IconDropbox />}
                                  {acc.provider === 'box' && <IconBox />}
                                  {acc.provider === 'yandex' && <IconYandex />}
                                  {acc.provider === 'pcloud' && <IconPCloud />}
                                  {acc.provider === 'telegram' && <IconTelegram />}
                                  {acc.provider === 'telegram_user' && <IconTelegram />}
                                  {acc.provider === 's3' && <IconSettings />}
                                  {acc.provider === 'webdav' && <IconCloud />}
                                  <span style={{ fontSize: '13px', fontWeight: '500' }}>{acc.displayName}</span>
                                </div>
                                <span style={{ fontSize: '11px', color: 'var(--md-sys-color-on-surface-variant)' }}>
                                  ({formatBytes(freeSpace)} free)
                                </span>
                              </div>

                              <div style={{ display: 'flex', gap: '6px' }}>
                                <button
                                  className="icon-btn"
                                  disabled={index === 0}
                                  onClick={() => handleMoveAccount(index, 'up')}
                                  style={{ padding: '4px', cursor: index === 0 ? 'not-allowed' : 'pointer', opacity: index === 0 ? 0.3 : 1 }}
                                >
                                  ▲
                                </button>
                                <button
                                  className="icon-btn"
                                  disabled={index === arr.length - 1}
                                  onClick={() => handleMoveAccount(index, 'down')}
                                  style={{ padding: '4px', cursor: index === arr.length - 1 ? 'not-allowed' : 'pointer', opacity: index === arr.length - 1 ? 0.3 : 1 }}
                                >
                                  ▼
                                </button>
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  )}
                </div>
              )}

              {storageTab === 'duplicates' && (
                <div>
                  <h3 style={{ fontSize: '16px', fontWeight: '600', marginBottom: '8px' }}>Duplicate Files Cleanup</h3>
                  <p style={{ fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)', marginBottom: '20px' }}>
                    Find and remove redundant file copies across your connected cloud accounts. Duplicate files are matched by name and exact content size.
                  </p>

                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
                    <button
                      type="button"
                      className="btn btn-filled"
                      disabled={duplicatesLoading}
                      onClick={scanDuplicateFiles}
                    >
                      {duplicatesLoading ? 'Scanning...' : 'Scan Now'}
                    </button>

                    {duplicateFiles.length > 0 && (
                      <button
                        type="button"
                        className="btn btn-filled"
                        style={{ backgroundColor: 'var(--md-sys-color-error)', color: 'var(--md-sys-color-on-error)' }}
                        disabled={selectedIDs.length === 0}
                        onClick={() => {
                          showConfirmDialog(
                            lang === 'id' ? 'Hapus Berkas Duplikat' : 'Delete Duplicate Files',
                            `Are you sure you want to permanently delete the ${selectedIDs.length} selected duplicate files?`,
                            async () => {
                              let successCount = 0;
                              let failCount = 0;
                              
                              for (const id of selectedIDs) {
                                try {
                                  const file = duplicateFiles.find(f => f.id === id);
                                  if (file) {
                                    await DeleteFile(file.id);
                                    successCount++;
                                  }
                                } catch (e) {
                                  failCount++;
                                }
                              }
                              
                              if (successCount > 0) {
                                showInfoDialog("Deletion Complete", `Successfully deleted ${successCount} duplicate files.${failCount > 0 ? ` (${failCount} failed)` : ''}`, "info");
                                setSelectedIDs([]);
                                scanDuplicateFiles();
                              } else {
                                showInfoDialog("Error", "Failed to delete selected duplicate files.");
                              }
                            },
                            { variant: 'danger', confirmLabel: 'Delete', cancelLabel: 'Cancel' }
                          );
                        }}
                      >
                        Delete Selected Duplicates ({selectedIDs.length})
                      </button>
                    )}
                  </div>

                  {duplicatesLoading ? (
                    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', marginTop: '60px', gap: '12px' }}>
                      <div className="loading-spinner" style={{ width: '32px', height: '32px', border: '3px solid var(--md-sys-color-surface-variant)', borderTop: '3px solid var(--md-sys-color-primary)', borderRadius: '50%', animation: 'spin 1s linear infinite' }}></div>
                      <span style={{ fontSize: '13px', color: 'var(--md-sys-color-on-surface-variant)' }}>Analyzing index tables for duplicate hashes...</span>
                    </div>
                  ) : duplicateFiles.length === 0 ? (
                    <div style={{ textAlign: 'center', padding: '60px 0', border: '1px dashed var(--md-sys-color-outline-variant)', borderRadius: '16px', color: 'var(--md-sys-color-on-surface-variant)' }}>
                      <svg width="40" height="40" viewBox="0 0 24 24" fill="currentColor" style={{ opacity: 0.7, marginBottom: '8px' }}><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/></svg>
                      <p style={{ fontSize: '14px', fontWeight: '500' }}>No duplicate files found</p>
                      <p style={{ fontSize: '11px', marginTop: '4px' }}>Click 'Scan Now' to run a synchronization search scan.</p>
                    </div>
                  ) : (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                      {(() => {
                        const groupsMap: Record<string, FileRecord[]> = {};
                        duplicateFiles.forEach(f => {
                          const key = `${f.name}-${f.size}`;
                          if (!groupsMap[key]) groupsMap[key] = [];
                          groupsMap[key].push(f);
                        });

                        return Object.entries(groupsMap).map(([key, items]) => {
                          const first = items[0];
                          return (
                            <div key={key} style={{ border: '1px solid var(--md-sys-color-outline-variant)', borderRadius: '12px', overflow: 'hidden', backgroundColor: 'var(--md-sys-color-surface-container)' }}>
                              <div style={{ padding: '12px 16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center', backgroundColor: 'var(--md-sys-color-surface-container-high)', borderBottom: '1px solid var(--md-sys-color-outline-variant)' }}>
                                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                                  <FileIcon name={first.name} />
                                  <span style={{ fontSize: '14px', fontWeight: '600', color: 'var(--md-sys-color-on-surface)', wordBreak: 'break-all' }}>{first.name}</span>
                                </div>
                                <span style={{ fontSize: '12px', fontWeight: '500', color: 'var(--md-sys-color-primary)' }}>{formatBytes(first.size)}</span>
                              </div>
                              <table className="files-table" style={{ width: '100%', borderCollapse: 'collapse', background: 'var(--md-sys-color-surface)' }}>
                                <tbody>
                                  {items.map(item => {
                                    const acc = accounts.find(a => a.id === item.accountId);
                                    return (
                                      <tr key={item.id} style={{ borderBottom: '1px solid var(--md-sys-color-outline-variant)' }}>
                                        <td style={{ width: '40px', padding: '12px' }}>
                                          <input
                                            type="checkbox"
                                            checked={selectedIDs.includes(item.id)}
                                            onChange={() => {
                                              if (selectedIDs.includes(item.id)) {
                                                setSelectedIDs(prev => prev.filter(id => id !== item.id));
                                              } else {
                                                setSelectedIDs(prev => [...prev, item.id]);
                                              }
                                            }}
                                            style={{ cursor: 'pointer' }}
                                          />
                                        </td>
                                        <td style={{ fontSize: '13px', color: 'var(--md-sys-color-on-surface)', padding: '12px' }}>
                                          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                            <span style={{ fontWeight: '500' }}>{acc ? acc.displayName : item.provider}</span>
                                            <span style={{ opacity: 0.6, fontSize: '11px' }}>({item.provider.toUpperCase()})</span>
                                          </div>
                                        </td>
                                        <td style={{ fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)', padding: '12px' }}>
                                          Modified: {formatDateTime(item.modifiedAt)}
                                        </td>
                                        <td style={{ width: '50px', padding: '12px', textAlign: 'right' }}>
                                          <button
                                            type="button"
                                            className="icon-btn"
                                            style={{ color: 'var(--md-sys-color-error)', cursor: 'pointer' }}
                                            onClick={() => {
                                              showConfirmDialog(
                                                lang === 'id' ? 'Hapus Salinan Duplikat' : 'Delete Duplicate Copy',
                                                `Are you sure you want to permanently delete this copy from ${acc?.displayName || item.provider}?`,
                                                async () => {
                                                  try {
                                                    await DeleteFile(item.id);
                                                    showInfoDialog("Deleted", "Duplicate copy deleted successfully.", "info");
                                                    scanDuplicateFiles();
                                                  } catch (e) {
                                                    showInfoDialog("Error", "Failed to delete: " + e);
                                                  }
                                                },
                                                { variant: 'danger', confirmLabel: 'Delete', cancelLabel: 'Cancel' }
                                              );
                                            }}
                                          >
                                            <IconDelete />
                                          </button>
                                        </td>
                                      </tr>
                                    );
                                  })}
                                </tbody>
                              </table>
                            </div>
                          );
                        });
                      })()}
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        )}

        {/* Settings tab */}
        {view === 'settings' && (
          <div className="explorer-body">
            <div className="explorer-header">
              <h2 style={{ fontSize: '20px', fontWeight: '500' }}>{t('settingsTitle')}</h2>
            </div>

            <div className="content-scroll" style={{ width: '100%', marginTop: '16px', paddingBottom: '40px' }}>
              {/* Settings Sub-Tabs Header (Pill Style matching Storage Menu) */}
              <div className="tab-buttons" style={{ display: 'flex', gap: '8px', marginBottom: '24px', flexWrap: 'wrap' }}>
                {[
                  { key: 'general', label: t('settingsTabGeneral') },
                  { key: 'vdisk', label: t('settingsTabVirtualDrive') },
                  { key: 'backup', label: t('settingsTabSyncTasks') },
                  { key: 'api', label: t('settingsTabApiConfig') }
                ].map(tab => {
                  const isActive = settingsTab === tab.key;
                  return (
                    <button
                      key={tab.key}
                      type="button"
                      className="btn"
                      onClick={() => setSettingsTab(tab.key as any)}
                      style={{
                        borderRadius: '100px',
                        padding: '8px 20px',
                        fontSize: '13px',
                        fontWeight: '500',
                        border: isActive ? 'none' : '1px solid var(--md-sys-color-outline-variant)',
                        backgroundColor: isActive ? 'var(--md-sys-color-primary-container)' : 'transparent',
                        color: isActive ? 'var(--md-sys-color-on-primary-container)' : 'var(--md-sys-color-on-surface-variant)',
                        cursor: 'pointer',
                        transition: 'all 0.2s'
                      }}
                    >
                      {tab.label}
                    </button>
                  );
                })}
              </div>

              {/* TAB 1: GENERAL SETTINGS */}
              {settingsTab === 'general' && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '24px', maxWidth: '640px' }}>
                  {/* Language Selection */}
                  <div className="form-group">
                    <label className="form-label" style={{ fontSize: '14px', fontWeight: '600' }}>{t('languageSetting')}</label>
                    <p style={{ fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)', marginBottom: '8px' }}>
                      {t('languageSettingDesc')}
                    </p>
                    <select
                      className="form-input"
                      style={{ height: '42px', padding: '0 12px', borderRadius: '8px' }}
                      value={lang}
                      onChange={(e) => handleLanguageChange(e.target.value as 'en' | 'id')}
                    >
                      <option value="en">{t('english')}</option>
                      <option value="id">{t('indonesian')}</option>
                    </select>
                  </div>

                  {/* Minimize to Tray Setting */}
                  <div className="form-group" style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                    <label className="form-label" style={{ fontSize: '14px', fontWeight: '600', display: 'flex', alignItems: 'center', gap: '10px', cursor: 'pointer' }}>
                      <input
                        type="checkbox"
                        checked={minToTray}
                        onChange={(e) => handleMinimizeTrayChange(e.target.checked)}
                        style={{ width: '18px', height: '18px', cursor: 'pointer' }}
                      />
                      <span>{t('minimizeToTraySetting')}</span>
                    </label>
                    <p style={{ fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)', paddingLeft: '28px', lineHeight: '1.4' }}>
                      {t('minimizeToTrayDesc')}
                    </p>
                  </div>

                  {/* Auto Startup Setting */}
                  <div className="form-group" style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                    <label className="form-label" style={{ fontSize: '14px', fontWeight: '600', display: 'flex', alignItems: 'center', gap: '10px', cursor: 'pointer' }}>
                      <input
                        type="checkbox"
                        checked={autoStartup}
                        onChange={async (e) => {
                          const enabled = e.target.checked;
                          setAutoStartup(enabled);
                          try {
                            // @ts-ignore
                            const res = await window.go?.main?.App?.SetStartup(enabled);
                            if (res && res.success === false) {
                              setAutoStartup(!enabled);
                              showInfoDialog("Error", "Failed to update startup setting: " + res.error);
                            } else {
                              showToast(enabled ? (lang === 'id' ? 'Startup otomatis diaktifkan' : 'Auto startup enabled') : (lang === 'id' ? 'Startup otomatis dinonaktifkan' : 'Auto startup disabled'));
                            }
                          } catch (err) {
                            setAutoStartup(!enabled);
                            showInfoDialog("Error", "Failed to update startup setting: " + err);
                          }
                        }}
                        style={{ width: '18px', height: '18px', cursor: 'pointer' }}
                      />
                      <span>{t('autoStartupSetting')}</span>
                    </label>
                    <p style={{ fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)', paddingLeft: '28px', lineHeight: '1.4' }}>
                      {t('autoStartupDesc')}
                    </p>
                  </div>
                </div>
              )}

              {/* TAB 2: VIRTUAL DRIVE ROUTER */}
              {settingsTab === 'vdisk' && (
                <div>
                  <VirtualDriveView theme={theme} lang={lang} />
                </div>
              )}

              {/* TAB 3: BACKUP & SYNC TASKS */}
              {settingsTab === 'backup' && (
                <div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px', flexWrap: 'wrap', gap: '16px' }}>
                    <div>
                      <h3 style={{ fontSize: '16px', fontWeight: '600', margin: 0 }}>{t('backupSettings')}</h3>
                      <p style={{ fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)', lineHeight: '1.4', margin: '4px 0 0' }}>
                        {t('autoBackupTasks')}
                      </p>
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <span style={{ fontSize: '12px', fontWeight: '500', color: 'var(--md-sys-color-on-surface-variant)' }}>
                          {t('backupIntervalSetting')}:
                        </span>
                        <select
                          className="form-input"
                          style={{ height: '36px', padding: '0 8px', fontSize: '12px', width: '130px', margin: 0 }}
                          value={backupInterval}
                          onChange={(e) => handleBackupIntervalChange(parseInt(e.target.value))}
                        >
                          <option value={60}>{t('minute1')}</option>
                          <option value={300}>{t('minutes5')}</option>
                          <option value={600}>{t('minutes10')}</option>
                          <option value={1800}>{t('minutes30')}</option>
                          <option value={3600}>{t('hour1')}</option>
                        </select>
                      </div>
                      <button
                        type="button"
                        className="btn btn-filled"
                        onClick={() => {
                          setBackupLocalPath('');
                          setBackupTargetFolderID('root');
                          setBackupAccountID('auto');
                          setBackupSyncMode('one-way');
                          setEditingSyncTask(null);
                          setModal({ type: 'backup-task' });
                        }}
                      >
                        {t('addBackupTask')}
                      </button>
                    </div>
                  </div>

                  {syncTasksLoading && <p style={{ fontSize: '13px', color: 'var(--md-sys-color-on-surface-variant)' }}>Loading tasks...</p>}
                  {!syncTasksLoading && syncTasks.length === 0 && (
                    <p style={{ fontSize: '13px', color: 'var(--md-sys-color-on-surface-variant)' }}>{t('backupEmpty')}</p>
                  )}
                  {!syncTasksLoading && syncTasks.length > 0 && (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', marginBottom: '28px' }}>
                      {syncTasks.map(task => (
                        <div key={task.id} className="dashboard-card" style={{ padding: '16px 20px', margin: 0, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <div style={{ display: 'flex', flexDirection: 'column', gap: '4px', minWidth: 0 }}>
                            <span style={{ fontSize: '14px', fontWeight: '600', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={task.localPath}>
                              {task.localPath}
                            </span>
                            <div style={{ display: 'flex', gap: '12px', flexWrap: 'wrap', fontSize: '11px', color: 'var(--md-sys-color-on-surface-variant)' }}>
                              <span>Mode: <strong>{task.syncMode === 'two-way' ? t('twoWay') : t('oneWay')}</strong></span>
                              <span>Destination: <strong>{task.accountId === 'auto' ? t('autoAllocate') : (accounts.find(a => a.id === task.accountId)?.displayName || task.accountId)}</strong></span>
                              {task.lastSync && <span>{t('lastSync')}: {task.lastSync}</span>}
                            </div>
                          </div>
                          <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flexShrink: 0 }}>
                            <label style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer' }}>
                              <input
                                type="checkbox"
                                checked={task.enabled}
                                onChange={(e) => handleToggleSyncTask(task.id, e.target.checked)}
                                style={{ width: '16px', height: '16px', cursor: 'pointer' }}
                              />
                              <span style={{ fontSize: '12px' }}>Active</span>
                            </label>
                            <button
                              type="button"
                              className="icon-btn"
                              style={{ color: 'var(--md-sys-color-primary)', cursor: 'pointer' }}
                              onClick={() => handleRunSyncTaskNow(task.id, task.localPath)}
                              title={lang === 'id' ? 'Cek & Sinkronkan Sekarang' : 'Check & Sync Now'}
                            >
                              <IconRefresh />
                            </button>
                            <button
                              type="button"
                              className="icon-btn"
                              style={{ color: 'var(--md-sys-color-primary)', cursor: 'pointer' }}
                              onClick={() => handleEditSyncTaskClick(task)}
                              title="Edit"
                            >
                              <IconRename />
                            </button>
                            <button
                              type="button"
                              className="icon-btn"
                              style={{ color: 'var(--md-sys-color-error)', cursor: 'pointer' }}
                              onClick={() => handleRemoveSyncTask(task.id, task.localPath)}
                              title={lang === 'id' ? 'Hapus Tugas Backup' : 'Delete Backup Task'}
                            >
                              <IconDelete />
                            </button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}

              {/* TAB 4: API CONFIGURATIONS */}
              {settingsTab === 'api' && (
                <div>
                  <h3 style={{ fontSize: '16px', fontWeight: '600', marginBottom: '8px' }}>{t('apiConfigs')}</h3>
                  <p style={{ fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)', marginBottom: '20px', lineHeight: '1.4' }}>
                    {t('apiConfigsDesc')}
                  </p>

                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(380px, 1fr))', gap: '16px' }}>
                    {/* Google Drive Card */}
                    <div className="dashboard-card" style={{ padding: '16px 20px', margin: 0 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '12px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                          <IconGoogleDrive />
                          <span style={{ fontWeight: '600', fontSize: '14px' }}>{t('googleCreds')}</span>
                          <span style={{
                            fontSize: '10px',
                            fontWeight: '600',
                            padding: '2px 8px',
                            borderRadius: '100px',
                            backgroundColor: isConfigured('google') ? 'rgba(15, 157, 88, 0.1)' : 'rgba(234, 67, 53, 0.1)',
                            color: isConfigured('google') ? '#0f9d58' : '#ea4335'
                          }}>
                            {isConfigured('google') ? t('configured') : t('notConfigured')}
                          </span>
                        </div>
                        <div style={{ display: 'flex', gap: '8px' }}>
                          <button className="btn btn-text" onClick={() => openCredentialsModal('google')}>{t('editConfig')}</button>
                          <button className="btn btn-text" style={{ fontSize: '12px' }} onClick={() => { setActiveGuide('google'); setShowGuideModal(true); }}>
                            {t('showGuide')}
                          </button>
                        </div>
                      </div>
                    </div>

                    {/* OneDrive Card */}
                    <div className="dashboard-card" style={{ padding: '16px 20px', margin: 0 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '12px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                          <IconOneDrive />
                          <span style={{ fontWeight: '600', fontSize: '14px' }}>{t('onedriveCreds')}</span>
                          <span style={{
                            fontSize: '10px',
                            fontWeight: '600',
                            padding: '2px 8px',
                            borderRadius: '100px',
                            backgroundColor: isConfigured('onedrive') ? 'rgba(15, 157, 88, 0.1)' : 'rgba(234, 67, 53, 0.1)',
                            color: isConfigured('onedrive') ? '#0f9d58' : '#ea4335'
                          }}>
                            {isConfigured('onedrive') ? t('configured') : t('notConfigured')}
                          </span>
                        </div>
                        <div style={{ display: 'flex', gap: '8px' }}>
                          <button className="btn btn-text" onClick={() => openCredentialsModal('onedrive')}>{t('editConfig')}</button>
                          <button className="btn btn-text" style={{ fontSize: '12px' }} onClick={() => { setActiveGuide('onedrive'); setShowGuideModal(true); }}>
                            {t('showGuide')}
                          </button>
                        </div>
                      </div>
                    </div>

                    {/* Dropbox Card */}
                    <div className="dashboard-card" style={{ padding: '16px 20px', margin: 0 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '12px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                          <IconDropbox />
                          <span style={{ fontWeight: '600', fontSize: '14px' }}>{t('dropboxCreds')}</span>
                          <span style={{
                            fontSize: '10px',
                            fontWeight: '600',
                            padding: '2px 8px',
                            borderRadius: '100px',
                            backgroundColor: isConfigured('dropbox') ? 'rgba(15, 157, 88, 0.1)' : 'rgba(234, 67, 53, 0.1)',
                            color: isConfigured('dropbox') ? '#0f9d58' : '#ea4335'
                          }}>
                            {isConfigured('dropbox') ? t('configured') : t('notConfigured')}
                          </span>
                        </div>
                        <div style={{ display: 'flex', gap: '8px' }}>
                          <button className="btn btn-text" onClick={() => openCredentialsModal('dropbox')}>{t('editConfig')}</button>
                          <button className="btn btn-text" style={{ fontSize: '12px' }} onClick={() => { setActiveGuide('dropbox'); setShowGuideModal(true); }}>
                            {t('showGuide')}
                          </button>
                        </div>
                      </div>
                    </div>

                    {/* Box Card */}
                    <div className="dashboard-card" style={{ padding: '16px 20px', margin: 0 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '12px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                          <IconBox />
                          <span style={{ fontWeight: '600', fontSize: '14px' }}>{t('boxCreds')}</span>
                          <span style={{
                            fontSize: '10px',
                            fontWeight: '600',
                            padding: '2px 8px',
                            borderRadius: '100px',
                            backgroundColor: isConfigured('box') ? 'rgba(15, 157, 88, 0.1)' : 'rgba(234, 67, 53, 0.1)',
                            color: isConfigured('box') ? '#0f9d58' : '#ea4335'
                          }}>
                            {isConfigured('box') ? t('configured') : t('notConfigured')}
                          </span>
                        </div>
                        <div style={{ display: 'flex', gap: '8px' }}>
                          <button className="btn btn-text" onClick={() => openCredentialsModal('box')}>{t('editConfig')}</button>
                          <button className="btn btn-text" style={{ fontSize: '12px' }} onClick={() => { setActiveGuide('box'); setShowGuideModal(true); }}>
                            {t('showGuide')}
                          </button>
                        </div>
                      </div>
                    </div>

                    {/* Yandex Disk Card */}
                    <div className="dashboard-card" style={{ padding: '16px 20px', margin: 0 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '12px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                          <IconYandex />
                          <span style={{ fontWeight: '600', fontSize: '14px' }}>{t('yandexCreds')}</span>
                          <span style={{
                            fontSize: '10px',
                            fontWeight: '600',
                            padding: '2px 8px',
                            borderRadius: '100px',
                            backgroundColor: isConfigured('yandex') ? 'rgba(15, 157, 88, 0.1)' : 'rgba(234, 67, 53, 0.1)',
                            color: isConfigured('yandex') ? '#0f9d58' : '#ea4335'
                          }}>
                            {isConfigured('yandex') ? t('configured') : t('notConfigured')}
                          </span>
                        </div>
                        <div style={{ display: 'flex', gap: '8px' }}>
                          <button className="btn btn-text" onClick={() => openCredentialsModal('yandex')}>{t('editConfig')}</button>
                          <button className="btn btn-text" style={{ fontSize: '12px' }} onClick={() => { setActiveGuide('yandex'); setShowGuideModal(true); }}>
                            {t('showGuide')}
                          </button>
                        </div>
                      </div>
                    </div>

                    {/* pCloud Card */}
                    <div className="dashboard-card" style={{ padding: '16px 20px', margin: 0 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '12px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                          <IconPCloud />
                          <span style={{ fontWeight: '600', fontSize: '14px' }}>{t('pcloudCreds')}</span>
                          <span style={{
                            fontSize: '10px',
                            fontWeight: '600',
                            padding: '2px 8px',
                            borderRadius: '100px',
                            backgroundColor: isConfigured('pcloud') ? 'rgba(15, 157, 88, 0.1)' : 'rgba(234, 67, 53, 0.1)',
                            color: isConfigured('pcloud') ? '#0f9d58' : '#ea4335'
                          }}>
                            {isConfigured('pcloud') ? t('configured') : t('notConfigured')}
                          </span>
                        </div>
                        <div style={{ display: 'flex', gap: '8px' }}>
                          <button className="btn btn-text" onClick={() => openCredentialsModal('pcloud')}>{t('editConfig')}</button>
                          <button className="btn btn-text" style={{ fontSize: '12px' }} onClick={() => { setActiveGuide('pcloud'); setShowGuideModal(true); }}>
                            {t('showGuide')}
                          </button>
                        </div>
                      </div>
                    </div>

                    {/* Telegram User API Card */}
                    <div className="dashboard-card" style={{ padding: '16px 20px', margin: 0 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '12px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                          <IconTelegram />
                          <span style={{ fontWeight: '600', fontSize: '14px' }}>{t('tgUserCreds')}</span>
                          <span style={{
                            fontSize: '10px',
                            fontWeight: '600',
                            padding: '2px 8px',
                            borderRadius: '100px',
                            backgroundColor: isConfigured('telegram_user') ? 'rgba(15, 157, 88, 0.1)' : 'rgba(234, 67, 53, 0.1)',
                            color: isConfigured('telegram_user') ? '#0f9d58' : '#ea4335'
                          }}>
                            {isConfigured('telegram_user') ? t('configured') : t('notConfigured')}
                          </span>
                        </div>
                        <div style={{ display: 'flex', gap: '8px' }}>
                          <button className="btn btn-text" onClick={() => openCredentialsModal('telegram_user')}>{t('editConfig')}</button>
                          <button className="btn btn-text" style={{ fontSize: '12px' }} onClick={() => { setActiveGuide('telegram_user'); setShowGuideModal(true); }}>
                            {t('showGuide')}
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Web Share Management tab */}
        {view === 'webshare' && (
          <div className="explorer-body">
            <div className="content-scroll" style={{ width: '100%', paddingBottom: '40px' }}>
              <WebShareManagement lang={lang} addToast={(msg) => showToast(msg)} />
            </div>
          </div>
        )}

        {/* About App tab */}
        {view === 'about' && (
          <div className="explorer-body">
            <div className="content-scroll" style={{ width: '100%', paddingBottom: '40px' }}>
              <AboutView lang={lang} />
            </div>
          </div>
        )}
      </main>

      {/* Floating Context Menu */}
      {contextMenu.visible && (
        <div className="context-menu" style={{ top: `${contextMenu.y}px`, left: `${contextMenu.x}px` }}>
          {contextMenu.file ? (
            view === 'trash' ? (
              <>
                <div className="context-item" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); handleRestoreFile(contextMenu.file!); }}>
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" style={{ marginRight: '8px' }}><path d="M12.5 8c-2.65 0-5.05.99-6.9 2.6L2 7v9h9l-3.62-3.62c1.39-1.16 3.16-1.88 5.12-1.88 3.54 0 6.55 2.31 7.6 5.5l2.37-.78C21.08 11.03 17.15 8 12.5 8z"/></svg>
                  <span>{t('restore')}</span>
                </div>
                <div className="context-item danger" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); handlePermanentDelete(contextMenu.file!); }}>
                  <IconDelete />
                  <span>{t('permanentlyDelete')}</span>
                </div>
              </>
            ) : (
              <>
                {/* Create Web Share Option */}
                <div className="context-item" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); openWebShareModal(contextMenu.file!); }}>
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
                  <span>{t('createWebShare')}</span>
                </div>
                {!contextMenu.file.isFolder && (
                <>
                  <div className="context-item" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); triggerFilePreview(contextMenu.file!); }}>
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/></svg>
                    <span>{t('preview')}</span>
                  </div>
                  <div className="context-item" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); handleDownload(contextMenu.file!); }}>
                    <IconDownload />
                    <span>{t('downloadFile')}</span>
                  </div>
                  <div className="context-item" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); openTransferModal(contextMenu.file!); }}>
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M16 17.01V10h-2v7.01h-3L15 21l4-3.99h-3zM9 3L5 6.99h3V14h2V6.99h3L9 3z"/></svg>
                    <span>{t('copyToAnotherCloud')}</span>
                  </div>
                  <div className="context-item" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); if (selectedIDs.length === 0 && contextMenu.file) setSelectedIDs([contextMenu.file.id]); setZipArchiveName('archive.zip'); setModal({ type: 'compress-zip' }); }}>
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M20 6h-8l-2-2H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm-6 8h-2v2h2v-2zm0-4h-2v2h2v-2zm0-4h-2v2h2V6z"/></svg>
                    <span>{t('compressToZip')}</span>
                  </div>
                  {contextMenu.file!.name.toLowerCase().endsWith('.zip') && (
                    <div className="context-item" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); handleExtractZip(contextMenu.file!); }}>
                      <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M19 9h-4V3H9v6H5l7 7 7-7zM5 18v2h14v-2H5z"/></svg>
                      <span>{t('extractHere')}</span>
                    </div>
                  )}
                </>
              )}
              {['google', 'onedrive', 'dropbox', 'box', 'yandex'].includes(contextMenu.file.provider.toLowerCase()) && (
                <div className="context-item" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); handleStarToggle(contextMenu.file!); }}>
                  <IconStar filled={contextMenu.file.starred} />
                  <span>{contextMenu.file.starred ? t('unstarFile') : t('starFile')}</span>
                </div>
              )}
              {contextMenu.file.provider !== 'virtual' && (
                <>
                  <div className="context-item has-submenu">
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                      <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M18 16.08c-.76 0-1.44.3-1.96.77L8.91 12.7c.05-.23.09-.46.09-.7s-.04-.47-.09-.7l7.05-4.11c.54.5 1.25.8 2.04.8 1.66 0 3-1.34 3-3s-1.34-3-3-3-3 1.34-3 3c0 .24.04.47.09.7L8.04 9.81C7.5 9.31 6.79 9 6 9c-1.66 0-3 1.34-3 3s1.34 3 3 3c.79 0 1.5-.3 2.04-.8l7.12 4.16c-.05.21-.08.43-.08.65 0 1.61 1.31 2.92 2.92 2.92 1.61 0 2.92-1.31 2.92-2.92s-1.31-2.92-2.92-2.92z"/></svg>
                      <span>{t('share')}</span>
                    </div>
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor" style={{ marginLeft: 'auto', color: 'var(--md-sys-color-on-surface-variant)' }}><path d="M8.59 16.59L13.17 12 8.59 7.41 10 6l6 6-6 6-1.41-1.41z"/></svg>
                    <div className="submenu">
                      <div className="context-item" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); openShareModal(contextMenu.file!); }}>
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M9 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0-6c1.1 0 2 .9 2 2s-.9 2-2 2-2-.9-2-2 .9-2 2-2zm0 7c-2.67 0-8 1.34-8 4v3h16v-3c0-2.66-5.33-4-8-4zm6 5H3v-.99c.2-.72 3.3-2.01 6-2.01s5.8 1.29 6 2v1zm-3-4.81c1.16-.85 2-2.04 2-3.44a5.955 5.955 0 0 0-2-4.44V3c2.76 0 5 2.24 5 5 0 1.95-1.11 3.63-2.73 4.46L12 11.19z"/></svg>
                        <span>{t('share')}</span>
                      </div>
                      <div className="context-item" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); handleCopyShareLink(contextMenu.file!); }}>
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M3.9 12c0-1.71 1.39-3.1 3.1-3.1h4V7H7c-2.76 0-5 2.24-5 5s2.24 5 5 5h4v-1.9H7c-1.71 0-3.1-1.39-3.1-3.1zM8 13h8v-2H8v2zm9-6h-4v1.9h4c1.71 0 3.1 1.39 3.1 3.1s-1.39 3.1-3.1 3.1h-4V17h4c2.76 0 5-2.24 5-5s-2.24-5-5-5z"/></svg>
                        <span>{t('copyLink')}</span>
                      </div>
                    </div>
                  </div>
                  <div className="context-item" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); handleOpenInCloud(contextMenu.file!); }}>
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M19 19H5V5h7V3H5c-1.11 0-2 .9-2 2v14c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2v-7h-2v7zM14 3v2h3.59l-9.83 9.83 1.41 1.41L19 6.41V10h2V3h-7z"/></svg>
                    <span>{t('openInDriveWeb')}</span>
                  </div>
                </>
              )}
              <div className="context-item" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); handleRename(contextMenu.file!); }}>
                <IconRename />
                <span>{t('renameItem')}</span>
              </div>
              <div className="context-item" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); setSelectedFile(contextMenu.file!); setDetailsSidebar(true); }}>
                <IconInfo />
                <span>{t('details')}</span>
              </div>
              <div className="context-item danger" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); handleDelete(contextMenu.file!); }}>
                <IconDelete />
                <span>
                  {['google', 'onedrive', 'dropbox', 'box', 'yandex', 'pcloud'].includes(contextMenu.file.provider.toLowerCase())
                    ? t('moveToTrash')
                    : t('permanentlyDelete')}
                </span>
              </div>
            </>
          ) ) : (
            <>
              <div className="context-item" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); setModal({ type: 'create-folder' }); }}>
                <IconFolder />
                <span>{t('newFolder')}</span>
              </div>
              <div className="context-item" onClick={() => { setContextMenu(prev => ({ ...prev, visible: false })); handleUploadFile(); }}>
                <IconDownload />
                <span>{t('fileUpload')}</span>
              </div>
            </>
          )}
        </div>
      )}

      {/* File Preview Modal */}
      {previewFile && (
        <div className="modal-overlay" style={{ backdropFilter: 'blur(16px)', backgroundColor: 'rgba(0, 0, 0, 0.75)', zIndex: 4000 }}>
          <div className="modal-content" style={{ width: '85vw', maxWidth: '1000px', height: '85vh', display: 'flex', flexDirection: 'column', padding: '24px', borderRadius: '24px', backgroundColor: 'var(--md-sys-color-surface)', border: '1px solid var(--md-sys-color-surface-container-high)' }}>
            
            {/* Header */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid var(--md-sys-color-surface-container-high)', paddingBottom: '16px', marginBottom: '16px' }}>
              <div>
                <h3 className="modal-header" style={{ margin: 0, fontSize: '18px', fontWeight: '600', wordBreak: 'break-all' }}>
                  {previewFile.name}
                </h3>
                {/* Account / Provider Details */}
                <div style={{ fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)', display: 'flex', alignItems: 'center', gap: '8px', marginTop: '4px' }}>
                  <span className={`provider-badge badge-${previewFile.provider}`}>{previewFile.provider}</span>
                  {(() => {
                    const acc = accounts.find(a => a.id === previewFile.accountId);
                    return acc ? (
                      <span>{acc.displayName} ({acc.email})</span>
                    ) : null;
                  })()}
                </div>
              </div>
              
              <div style={{ display: 'flex', gap: '8px' }}>
                <button className="icon-btn" onClick={() => handleDownload(previewFile)} title={t('downloadFile')} style={{ border: '1px solid var(--md-sys-color-outline-variant)' }}>
                  <IconDownload />
                </button>
                <button className="icon-btn" onClick={() => setPreviewFile(null)} title={t('cancel')} style={{ border: '1px solid var(--md-sys-color-outline-variant)' }}>
                  <IconClose />
                </button>
              </div>
            </div>

            {/* Content Area */}
            <div style={{ flexGrow: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden', position: 'relative' }}>
              {previewLoading && (
                <div style={{ textAlign: 'center' }}>
                  <div className="icon-btn spinning" style={{ width: '48px', height: '48px', margin: '0 auto 16px' }}>
                    <IconRefresh />
                  </div>
                  <p style={{ fontSize: '14px', color: 'var(--md-sys-color-on-surface-variant)' }}>{t('loadingPreview')}</p>
                </div>
              )}

              {previewError && (
                <div style={{ textAlign: 'center', padding: '24px', color: 'var(--md-sys-color-error)' }}>
                  <svg width="48" height="48" viewBox="0 0 24 24" fill="currentColor" style={{ margin: '0 auto 16px', display: 'block' }}><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>
                  <p style={{ fontWeight: '500', marginBottom: '8px' }}>{t('previewError')}</p>
                  <p style={{ fontSize: '13px', color: 'var(--md-sys-color-on-surface-variant)', wordBreak: 'break-all' }}>{previewError}</p>
                </div>
              )}

              {previewData && (
                <div style={{ width: '100%', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'auto' }}>
                  {previewData.base64 && (previewData.ext === '.jpg' || previewData.ext === '.jpeg' || previewData.ext === '.png' || previewData.ext === '.gif' || previewData.ext === '.webp' || previewData.ext === '.bmp' || previewData.ext === '.ico' || previewData.ext === '.svg') && (
                    <img src={previewData.base64} alt={previewFile.name} style={{ maxWidth: '100%', maxHeight: '100%', objectFit: 'contain', borderRadius: '8px', boxShadow: 'var(--shadow-2)' }} />
                  )}
                  {previewData.base64 && previewData.ext === '.pdf' && (
                    <embed src={previewData.base64} type="application/pdf" width="100%" height="100%" style={{ borderRadius: '8px', border: 'none' }} />
                  )}
                  {previewData.base64 && (previewData.ext === '.mp3' || previewData.ext === '.wav' || previewData.ext === '.ogg' || previewData.ext === '.m4a' || previewData.ext === '.flac' || previewData.ext === '.aac') && (
                    <div style={{ width: '80%', textAlign: 'center' }}>
                      <audio controls src={previewData.base64} style={{ width: '100%' }}>
                        Your browser does not support the audio element.
                      </audio>
                      <p style={{ marginTop: '16px', color: 'var(--md-sys-color-on-surface-variant)' }}>{previewFile.name}</p>
                    </div>
                  )}
                  {previewData.base64 && (previewData.ext === '.mp4' || previewData.ext === '.webm' || previewData.ext === '.ogv' || previewData.ext === '.mov' || previewData.ext === '.mkv') && (
                    <video controls src={previewData.base64} style={{ maxWidth: '100%', maxHeight: '100%', borderRadius: '8px' }}>
                      Your browser does not support the video element.
                    </video>
                  )}
                  {previewData.text !== undefined && (
                    <pre style={{ width: '100%', height: '100%', textAlign: 'left', backgroundColor: 'var(--md-sys-color-surface-container)', color: 'var(--md-sys-color-on-surface)', padding: '20px', borderRadius: '12px', overflow: 'auto', fontFamily: 'Consolas, Monaco, monospace', fontSize: '13px', whiteSpace: 'pre-wrap', border: '1px solid var(--md-sys-color-surface-container-high)' }}>
                      {previewData.text}
                    </pre>
                  )}
                </div>
              )}
            </div>

          </div>
        </div>
      )}

      {/* Floating Transfer Notification Drawer */}
      {transfers.length > 0 && (
        <div className="upload-drawer">
          <div className="upload-drawer-header">
            <span>{t('transfers')}</span>
            <button className="icon-btn" style={{ width: '28px', height: '28px' }} onClick={() => setTransfers([])} title="Clear notifications">
              <IconClose />
            </button>
          </div>
          <div className="upload-drawer-list">
            {transfers.map(t_item => (
              <div key={t_item.id} className="upload-item" style={{ display: 'flex', flexDirection: 'column', gap: '6px', alignItems: 'stretch' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '12px' }}>
                  <span className="upload-item-name" title={t_item.name} style={{ flexGrow: 1, minWidth: 0 }}>{t_item.name}</span>
                  <span className={`upload-item-status ${t_item.status}`} style={{ flexShrink: 0 }}>
                    {t_item.status === 'started' && (t_item.type === 'upload' ? t('uploading') : t_item.type === 'transfer' ? 'Transferring...' : t('downloading'))}
                    {t_item.status === 'uploading' && `${t('uploading')} ${typeof t_item.progress === 'number' ? `${t_item.progress}%` : ''}`}
                    {t_item.status === 'downloading' && `${t('downloading')} ${typeof t_item.progress === 'number' ? `${t_item.progress}%` : ''}`}
                    {t_item.status === 'transferring' && `Transferring... ${typeof t_item.progress === 'number' ? `${t_item.progress}%` : ''}`}
                    {t_item.status === 'completed' && t('completed')}
                    {t_item.status === 'failed' && t('failed')}
                  </span>
                </div>
                {typeof t_item.progress === 'number' && t_item.status !== 'failed' && (
                  <div style={{ width: '100%', height: '6px', borderRadius: '999px', background: 'var(--md-sys-color-surface-container-high)', overflow: 'hidden' }}>
                    <div style={{ width: `${Math.max(0, Math.min(100, t_item.progress))}%`, height: '100%', borderRadius: '999px', background: 'var(--md-sys-color-primary)' }} />
                  </div>
                )}
                {t_item.error && <div style={{ fontSize: '10px', color: 'var(--md-sys-color-error)' }}>{t_item.error}</div>}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Modals manager */}
      {modal?.type === 'share' && shareFile && (
        <div className="modal-overlay">
          <div className="modal-content" style={{ maxWidth: '480px', width: '90vw', padding: '24px', borderRadius: '28px', backgroundColor: 'var(--md-sys-color-surface)', border: '1px solid var(--md-sys-color-outline-variant)' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
              <h3 className="modal-header" style={{ margin: 0, fontSize: '20px', fontWeight: '500' }}>
                {t('shareTitle').replace('{name}', shareFile.name)}
              </h3>
              <span className={`provider-badge badge-${shareFile.provider}`} style={{ fontSize: '11px', textTransform: 'uppercase' }}>
                {shareFile.provider}
              </span>
            </div>

            {shareLoading && (
              <div style={{ padding: '40px 0', textAlign: 'center', color: 'var(--md-sys-color-primary)', fontSize: '14px', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '12px' }}>
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="rotating-loader" style={{ animation: 'spin 1s linear infinite' }}>
                  <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/>
                </svg>
                <span>{t('fetchingSharingOptions')}</span>
              </div>
            )}

            {!shareLoading && (
              <>
                <form onSubmit={handleAddPermission} style={{ display: 'flex', gap: '8px', marginBottom: '20px' }}>
                  <input
                    type="email"
                    className="form-input"
                    placeholder={t('addPeoplePlaceholder')}
                    required
                    value={shareEmail}
                    onChange={(e) => setShareEmail(e.target.value)}
                    style={{ flexGrow: 1 }}
                  />
                  <select
                    className="form-input"
                    value={shareRole}
                    onChange={(e) => setShareRole(e.target.value as 'reader' | 'writer')}
                    style={{ width: '100px', flexShrink: 0 }}
                  >
                    <option value="reader">{t('viewer')}</option>
                    <option value="writer">{t('editor')}</option>
                  </select>
                  <button type="submit" className="btn btn-filled" style={{ borderRadius: '8px', padding: '0 16px', height: '44px', flexShrink: 0 }}>
                    {t('share')}
                  </button>
                </form>

                <h4 style={{ fontSize: '13px', fontWeight: '600', marginBottom: '8px', color: 'var(--md-sys-color-on-surface)' }}>{t('peopleWithAccess')}</h4>
                <div style={{ maxHeight: '160px', overflowY: 'auto', marginBottom: '20px', display: 'flex', flexDirection: 'column', gap: '8px', paddingRight: '4px' }}>
                  {sharePermissions.filter(p => p.type === 'user' || p.type === 'group' || p.role === 'owner').map(p => (
                    <div key={p.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 12px', borderRadius: '8px', backgroundColor: 'var(--md-sys-color-surface-container)' }}>
                      <div style={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
                        <span style={{ fontSize: '13px', fontWeight: '500', color: 'var(--md-sys-color-on-surface)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {p.displayName || p.emailAddress || 'User'}
                        </span>
                        {p.emailAddress && (
                          <span style={{ fontSize: '11px', color: 'var(--md-sys-color-on-surface-variant)' }}>
                            {p.emailAddress}
                          </span>
                        )}
                      </div>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <span style={{ fontSize: '11px', fontWeight: '600', textTransform: 'capitalize', padding: '3px 8px', borderRadius: '100px', backgroundColor: p.role === 'owner' ? 'var(--md-sys-color-primary-container)' : 'var(--md-sys-color-surface-container-high)', color: p.role === 'owner' ? 'var(--md-sys-color-on-primary-container)' : 'var(--md-sys-color-on-surface-variant)' }}>
                          {p.role === 'reader' ? t('viewer') : p.role === 'writer' ? t('editor') : p.role}
                        </span>
                        {p.role !== 'owner' && (
                          <button
                            className="icon-btn"
                            style={{ color: 'var(--md-sys-color-error)', width: '28px', height: '28px' }}
                            onClick={() => handleDeletePermission(p.id)}
                            title="Remove access"
                          >
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/></svg>
                          </button>
                        )}
                      </div>
                    </div>
                  ))}
                  {sharePermissions.filter(p => p.type === 'user' || p.type === 'group' || p.role === 'owner').length === 0 && (
                    <div style={{ fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)', textAlign: 'center', padding: '12px' }}>
                      {t('noSpecificUsers')}
                    </div>
                  )}
                </div>

                <h4 style={{ fontSize: '13px', fontWeight: '600', marginBottom: '8px', color: 'var(--md-sys-color-on-surface)' }}>{t('generalAccess')}</h4>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', padding: '12px', borderRadius: '12px', backgroundColor: 'var(--md-sys-color-surface-container-high)', marginBottom: '20px' }}>
                  <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
                    <div style={{ width: '36px', height: '36px', borderRadius: '50%', backgroundColor: shareGeneralAccess === 'anyone' ? 'var(--md-sys-color-primary-container)' : 'var(--md-sys-color-surface-container-highest)', color: shareGeneralAccess === 'anyone' ? 'var(--md-sys-color-on-primary-container)' : 'var(--md-sys-color-on-surface-variant)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                      {shareGeneralAccess === 'anyone' ? (
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/></svg>
                      ) : (
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zm-6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm3.1-9H8.9V6c0-1.71 1.39-3.1 3.1-3.1 1.71 0 3.1 1.39 3.1 3.1v2z"/></svg>
                      )}
                    </div>
                    <div style={{ display: 'flex', flexDirection: 'column', flexGrow: 1, minWidth: 0 }}>
                      <select
                        className="form-input"
                        value={shareGeneralAccess}
                        onChange={(e) => handleGeneralAccessChange(e.target.value)}
                        style={{ height: '32px', padding: '0', fontSize: '13px', border: 'none', background: 'transparent', color: 'var(--md-sys-color-on-surface)', fontWeight: '600', cursor: 'pointer', width: 'auto' }}
                      >
                        <option value="restricted">{t('restricted')}</option>
                        <option value="anyone">{t('anyoneWithLink')}</option>
                      </select>
                      <span style={{ fontSize: '11px', color: 'var(--md-sys-color-on-surface-variant)', paddingLeft: '2px' }}>
                        {shareGeneralAccess === 'restricted'
                          ? t('restrictedDesc')
                          : t('anyoneWithLinkDesc')}
                      </span>
                    </div>
                    {shareGeneralAccess === 'anyone' && (
                      <select
                        className="form-input"
                        value={shareGeneralRole}
                        onChange={(e) => handleGeneralAccessChange('anyone', e.target.value as 'reader' | 'writer')}
                        style={{ width: '90px', height: '36px', padding: '0 8px', fontSize: '12px', flexShrink: 0 }}
                      >
                        <option value="reader">{t('viewer')}</option>
                        <option value="writer">{t('editor')}</option>
                      </select>
                    )}
                  </div>
                </div>
              </>
            )}

            <div className="modal-footer" style={{ marginTop: '16px', display: 'flex', justifyContent: 'space-between', width: '100%', padding: 0 }}>
              <button
                type="button"
                className="btn btn-text"
                onClick={async () => {
                  try {
                    const url = await GetFileWebURL(shareFile.id);
                    await navigator.clipboard.writeText(url);
                    showToast("Link copied");
                  } catch (err) {
                    showInfoDialog("Error", "Could not copy link: " + err);
                  }
                }}
                disabled={shareLoading}
                style={{ display: 'flex', alignItems: 'center', gap: '6px', padding: '10px 12px' }}
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M3.9 12c0-1.71 1.39-3.1 3.1-3.1h4V7H7c-2.76 0-5 2.24-5 5s2.24 5 5 5h4v-1.9H7c-1.71 0-3.1-1.39-3.1-3.1zM8 13h8v-2H8v2zm9-6h-4v1.9h4c1.71 0 3.1 1.39 3.1 3.1s-1.39 3.1-3.1 3.1h-4V17h4c2.76 0 5-2.24 5-5s-2.24-5-5-5z"/></svg>
                {t('copyLink')}
              </button>
              <button type="button" className="btn btn-filled" onClick={() => { setModal(null); setShareFile(null); }} style={{ padding: '10px 24px' }}>
                {t('done')}
              </button>
            </div>
          </div>
        </div>
      )}

      {modal?.type === 'transfer-file' && transferFile && (
        <div className="modal-overlay">
          <form className="modal-content" onSubmit={handleTransferSubmit}>
            <h3 className="modal-header">Copy to another cloud</h3>
            <p style={{ fontSize: '13px', color: 'var(--md-sys-color-on-surface-variant)', marginBottom: '20px' }}>
              Select target cloud account and folder directory to copy <strong>{transferFile.name}</strong> directly.
            </p>

            <div className="form-group">
              <label className="form-label">Destination Account</label>
              <select
                className="form-input"
                value={selectedDestAccountID}
                required
                onChange={(e) => setSelectedDestAccountID(e.target.value)}
              >
                <option value="" disabled>Select target account...</option>
                {accounts.filter(a => a.active).map(acc => (
                  <option key={acc.id} value={acc.id}>
                    {acc.displayName} ({acc.provider.toUpperCase()})
                  </option>
                ))}
              </select>
            </div>

            <div className="form-group">
              <label className="form-label">Destination Virtual Folder</label>
              <select
                className="form-input"
                value={selectedDestFolderID}
                required
                onChange={(e) => setSelectedDestFolderID(e.target.value)}
              >
                {virtualFolders.map(folder => (
                  <option key={folder.id} value={folder.id}>
                    {folder.id === 'root' ? '/ (Main Storage root)' : `/${folder.name}`}
                  </option>
                ))}
              </select>
            </div>

            <div className="modal-footer">
              <button type="button" className="btn btn-text" disabled={transferLoading} onClick={() => { setModal(null); setTransferFile(null); }}>{t('cancel')}</button>
              <button type="submit" className="btn btn-filled" disabled={transferLoading || !selectedDestAccountID}>
                {transferLoading ? 'Starting...' : 'Copy File'}
              </button>
            </div>
          </form>
        </div>
      )}

      {modal?.type === 'remote-upload' && (
        <div className="modal-overlay">
          <form className="modal-content" onSubmit={handleRemoteUploadSubmit}>
            <h3 className="modal-header">Remote URL Upload</h3>
            <p style={{ fontSize: '13px', color: 'var(--md-sys-color-on-surface-variant)', marginBottom: '20px' }}>
              Download a file directly from a URL to your selected cloud storage folder.
            </p>

            <div className="form-group">
              <label className="form-label">Direct Download URL</label>
              <input
                type="url"
                className="form-input"
                placeholder="https://example.com/archive.zip"
                required
                autoFocus
                value={remoteUploadURL}
                onChange={(e) => setRemoteUploadURL(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">Target Cloud Account</label>
              <select
                className="form-input"
                value={remoteUploadAccountID}
                required
                onChange={(e) => setRemoteUploadAccountID(e.target.value)}
              >
                <option value="" disabled>Select target account...</option>
                {accounts.filter(a => a.active).map(acc => (
                  <option key={acc.id} value={acc.id}>
                    {acc.displayName} ({acc.provider.toUpperCase()})
                  </option>
                ))}
              </select>
            </div>

            <div className="modal-footer">
              <button type="button" className="btn btn-text" disabled={transferLoading} onClick={() => { setModal(null); setRemoteUploadURL(''); }}>{t('cancel')}</button>
              <button type="submit" className="btn btn-filled" disabled={transferLoading || !remoteUploadURL.trim() || !remoteUploadAccountID}>
                {transferLoading ? 'Starting...' : 'Upload URL'}
              </button>
            </div>
          </form>
        </div>
      )}

      {modal?.type === 'compress-zip' && (
        <div className="modal-overlay">
          <form className="modal-content" onSubmit={handleCompressFilesSubmit}>
            <h3 className="modal-header">Compress Selected to ZIP</h3>
            <p style={{ fontSize: '13px', color: 'var(--md-sys-color-on-surface-variant)', marginBottom: '20px' }}>
              Zip up the {selectedIDs.length} selected files directly in the cloud folder.
            </p>

            <div className="form-group">
              <label className="form-label">Archive File Name</label>
              <input
                type="text"
                className="form-input"
                placeholder="archive.zip"
                required
                autoFocus
                value={zipArchiveName}
                onChange={(e) => setZipArchiveName(e.target.value)}
              />
            </div>

            <div className="modal-footer">
              <button type="button" className="btn btn-text" disabled={transferLoading} onClick={() => { setModal(null); setZipArchiveName(''); setSelectedIDs([]); }}>{t('cancel')}</button>
              <button type="submit" className="btn btn-filled" disabled={transferLoading || !zipArchiveName.trim()}>
                {transferLoading ? 'Creating ZIP...' : 'Compress'}
              </button>
            </div>
          </form>
        </div>
      )}

      {modal?.type === 'create-folder' && (
        <div className="modal-overlay">
          <form className="modal-content" onSubmit={handleCreateFolderSubmit}>
            <h3 className="modal-header">{t('createFolderHeader')}</h3>
            <div className="form-group">
              <label className="form-label">{t('folderNameLabel')}</label>
              <input
                type="text"
                className="form-input"
                autoFocus
                required
                value={folderNameInput}
                onChange={(e) => setFolderNameInput(e.target.value)}
              />
            </div>
            <div className="modal-footer">
              <button type="button" className="btn btn-text" onClick={() => { setModal(null); setFolderNameInput(''); }}>{t('cancel')}</button>
              <button type="submit" className="btn btn-filled">{t('create')}</button>
            </div>
          </form>
        </div>
      )}

      {modal?.type === 'add-account' && (
        <div className="modal-overlay">
          <div className="modal-content" style={{ width: '520px', maxHeight: '90vh', overflowY: 'auto' }}>
            <h3 className="modal-header">{t('selectProviderHeader')}</h3>
            <p style={{ fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)', marginBottom: '16px' }}>
              {t('selectProviderDesc')}
            </p>

            {(!isProviderConfigured('google') || 
              !isProviderConfigured('onedrive') || 
              !isProviderConfigured('dropbox') || 
              !isProviderConfigured('box') || 
              !isProviderConfigured('yandex') || 
              !isProviderConfigured('pcloud')) && (
              <div className="alert-panel" style={{ color: 'var(--md-sys-color-error)', borderColor: 'var(--md-sys-color-error)', marginBottom: '12px', fontSize: '12px', lineHeight: '1.4' }}>
                {t('needSetupWarning')}
              </div>
            )}

            {!isProviderConfigured('telegram_user') && (
              <div className="alert-panel" style={{ color: 'var(--md-sys-color-error)', borderColor: 'var(--md-sys-color-error)', marginBottom: '16px', fontSize: '12px', lineHeight: '1.4' }}>
                {t('telegramNeedApiWarning')}
              </div>
            )}

            <div style={{ marginBottom: '20px' }}>
              <h4 style={{ fontSize: '13px', fontWeight: '600', marginBottom: '10px', color: 'var(--md-sys-color-primary)', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
                {t('oauthProvidersHeader')}
              </h4>
              <div className="providers-picker" style={{ gridTemplateColumns: 'repeat(3, 1fr)', gap: '12px' }}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                  <button className="provider-picker-btn" disabled={!isProviderConfigured('google')} onClick={() => handleLinkAccount('google')} style={{ width: '100%', padding: '16px 8px' }}>
                    <IconGoogleDrive />
                    <span>Google Drive</span>
                  </button>
                  <button className="btn btn-text" style={{ fontSize: '10px', height: 'auto', padding: '4px' }} onClick={() => { setActiveGuide('google'); setShowGuideModal(true); }}>{t('showGuide')}</button>
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                  <button className="provider-picker-btn" disabled={!isProviderConfigured('onedrive')} onClick={() => handleLinkAccount('onedrive')} style={{ width: '100%', padding: '16px 8px' }}>
                    <IconOneDrive />
                    <span>OneDrive</span>
                  </button>
                  <button className="btn btn-text" style={{ fontSize: '10px', height: 'auto', padding: '4px' }} onClick={() => { setActiveGuide('onedrive'); setShowGuideModal(true); }}>{t('showGuide')}</button>
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                  <button className="provider-picker-btn" disabled={!isProviderConfigured('dropbox')} onClick={() => handleLinkAccount('dropbox')} style={{ width: '100%', padding: '16px 8px' }}>
                    <IconDropbox />
                    <span>Dropbox</span>
                  </button>
                  <button className="btn btn-text" style={{ fontSize: '10px', height: 'auto', padding: '4px' }} onClick={() => { setActiveGuide('dropbox'); setShowGuideModal(true); }}>{t('showGuide')}</button>
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                  <button className="provider-picker-btn" disabled={!isProviderConfigured('box')} onClick={() => handleLinkAccount('box')} style={{ width: '100%', padding: '16px 8px' }}>
                    <IconBox />
                    <span>Box</span>
                  </button>
                  <button className="btn btn-text" style={{ fontSize: '10px', height: 'auto', padding: '4px' }} onClick={() => { setActiveGuide('box'); setShowGuideModal(true); }}>{t('showGuide')}</button>
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                  <button className="provider-picker-btn" disabled={!isProviderConfigured('yandex')} onClick={() => handleLinkAccount('yandex')} style={{ width: '100%', padding: '16px 8px' }}>
                    <IconYandex />
                    <span>Yandex Disk</span>
                  </button>
                  <button className="btn btn-text" style={{ fontSize: '10px', height: 'auto', padding: '4px' }} onClick={() => { setActiveGuide('yandex'); setShowGuideModal(true); }}>{t('showGuide')}</button>
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                  <button className="provider-picker-btn" disabled={!isProviderConfigured('pcloud')} onClick={() => handleLinkAccount('pcloud')} style={{ width: '100%', padding: '16px 8px' }}>
                    <IconPCloud />
                    <span>pCloud</span>
                  </button>
                  <button className="btn btn-text" style={{ fontSize: '10px', height: 'auto', padding: '4px' }} onClick={() => { setActiveGuide('pcloud'); setShowGuideModal(true); }}>{t('showGuide')}</button>
                </div>
              </div>
            </div>

            <div style={{ marginBottom: '20px' }}>
              <h4 style={{ fontSize: '13px', fontWeight: '600', marginBottom: '10px', color: 'var(--md-sys-color-primary)', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
                {t('directProvidersHeader')}
              </h4>
              <div className="providers-picker" style={{ gridTemplateColumns: 'repeat(3, 1fr)', gap: '12px' }}>
                <button className="provider-picker-btn" onClick={() => handleLinkAccount('telegram')}>
                  <IconTelegram />
                  <span>Telegram Bot</span>
                </button>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                  <button className="provider-picker-btn" disabled={!isProviderConfigured('telegram_user')} onClick={() => handleLinkAccount('telegram_user')} style={{ width: '100%', padding: '16px 8px' }}>
                    <IconTelegram />
                    <span>Telegram User</span>
                  </button>
                  <button className="btn btn-text" style={{ fontSize: '10px', height: 'auto', padding: '4px' }} onClick={() => { setActiveGuide('telegram_user'); setShowGuideModal(true); }}>{t('showGuide')}</button>
                </div>
                <button className="provider-picker-btn" onClick={() => handleLinkAccount('s3')}>
                  <IconSettings />
                  <span>S3 Compatible</span>
                </button>
                <button className="provider-picker-btn" onClick={() => handleLinkAccount('mega')}>
                  <IconMega />
                  <span>MEGA</span>
                </button>
                <button className="provider-picker-btn" onClick={() => handleLinkAccount('koofr')}>
                  <IconKoofr />
                  <span>Koofr</span>
                </button>
                <button className="provider-picker-btn" onClick={() => handleLinkAccount('mediafire')}>
                  <IconMediaFire />
                  <span>MediaFire</span>
                </button>
                <button className="provider-picker-btn" onClick={() => handleLinkAccount('fourshared')}>
                  <Icon4Shared />
                  <span>4Shared</span>
                </button>
                <button className="provider-picker-btn" onClick={() => handleLinkAccount('b2')}>
                  <IconB2 />
                  <span>Backblaze B2</span>
                </button>
                <button className="provider-picker-btn" onClick={() => handleLinkAccount('smb')}>
                  <IconSmb />
                  <span>Windows Share (SMB)</span>
                </button>
                <button className="provider-picker-btn" onClick={() => handleLinkAccount('ftp')}>
                  <IconFtp />
                  <span>FTP Server</span>
                </button>
                <button className="provider-picker-btn" onClick={() => handleLinkAccount('sftp')}>
                  <IconSftp />
                  <span>SFTP (SSH)</span>
                </button>
                <button className="provider-picker-btn" onClick={() => handleLinkAccount('webdav')}>
                  <IconCloud />
                  <span>WebDAV</span>
                </button>
              </div>
            </div>

            <div className="modal-footer" style={{ borderTop: '1px solid var(--md-sys-color-surface-container-high)', paddingTop: '12px', marginTop: '16px' }}>
              <button className="btn btn-text" onClick={() => setModal(null)}>{t('cancel')}</button>
            </div>
          </div>
        </div>
      )}

      {modal?.type === 'mega' && (
        <div className="modal-overlay">
          <form className="modal-content" style={{ width: '440px' }} onSubmit={async (e) => {
            e.preventDefault();
            setMegaLoading(true);
            setMegaError('');
            try {
              // @ts-ignore
              await window.go?.main?.App?.AddMegaAccount(megaEmail, megaPassword);
              showToast(lang === 'id' ? 'Akun MEGA berhasil terhubung!' : 'MEGA account connected successfully!');
              setModal(null);
              setMegaEmail('');
              setMegaPassword('');
              fetchAccounts();
            } catch (err: any) {
              setMegaError(err?.message || String(err));
            } finally {
              setMegaLoading(false);
            }
          }}>
            <h3 className="modal-header">{t('connectMega')}</h3>
            
            {megaError && (
              <div className="alert-panel" style={{ color: 'var(--md-sys-color-error)', borderColor: 'var(--md-sys-color-error)', marginBottom: '16px' }}>
                {megaError}
              </div>
            )}

            <div className="form-group">
              <label className="form-label">{t('megaEmailLabel')}</label>
              <input
                type="email"
                className="form-input"
                required
                placeholder="name@example.com"
                value={megaEmail}
                onChange={(e) => setMegaEmail(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('megaPasswordLabel')}</label>
              <input
                type="password"
                className="form-input"
                required
                value={megaPassword}
                onChange={(e) => setMegaPassword(e.target.value)}
              />
            </div>

            <div className="modal-footer">
              <button type="button" className="btn btn-text" disabled={megaLoading} onClick={() => setModal(null)}>{t('cancel')}</button>
              <button type="submit" className="btn btn-filled" disabled={megaLoading}>
                {megaLoading ? t('connecting') : t('create')}
              </button>
            </div>
          </form>
        </div>
      )}

      {modal?.type === 'koofr' && (
        <div className="modal-overlay">
          <form className="modal-content" style={{ width: '440px' }} onSubmit={async (e) => {
            e.preventDefault();
            setKoofrLoading(true);
            setKoofrError('');
            try {
              // @ts-ignore
              await window.go?.main?.App?.AddKoofrAccount(koofrUser, koofrPass);
              showToast(lang === 'id' ? 'Akun Koofr berhasil terhubung!' : 'Koofr account connected successfully!');
              setModal(null);
              setKoofrUser('');
              setKoofrPass('');
              fetchAccounts();
            } catch (err: any) {
              setKoofrError(err?.message || String(err));
            } finally {
              setKoofrLoading(false);
            }
          }}>
            <h3 className="modal-header">{t('connectKoofr')}</h3>
            
            {koofrError && (
              <div className="alert-panel" style={{ color: 'var(--md-sys-color-error)', borderColor: 'var(--md-sys-color-error)', marginBottom: '16px' }}>
                {koofrError}
              </div>
            )}

            <div className="form-group">
              <label className="form-label">{t('koofrUserLabel')}</label>
              <input
                type="text"
                className="form-input"
                required
                placeholder="user@example.com"
                value={koofrUser}
                onChange={(e) => setKoofrUser(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('koofrPassLabel')}</label>
              <input
                type="password"
                className="form-input"
                required
                placeholder="App password generated from Koofr settings"
                value={koofrPass}
                onChange={(e) => setKoofrPass(e.target.value)}
              />
            </div>

            <div className="modal-footer">
              <button type="button" className="btn btn-text" disabled={koofrLoading} onClick={() => setModal(null)}>{t('cancel')}</button>
              <button type="submit" className="btn btn-filled" disabled={koofrLoading}>
                {koofrLoading ? t('connecting') : t('create')}
              </button>
            </div>
          </form>
        </div>
      )}

      {modal?.type === 'mediafire' && (
        <div className="modal-overlay">
          <form className="modal-content" style={{ width: '440px' }} onSubmit={async (e) => {
            e.preventDefault();
            setMediafireLoading(true);
            setMediafireError('');
            try {
              // @ts-ignore
              await window.go?.main?.App?.AddMediaFireAccount(mediafireEmail, mediafirePassword);
              showToast(lang === 'id' ? 'Akun MediaFire berhasil terhubung!' : 'MediaFire account connected successfully!');
              setModal(null);
              setMediafireEmail('');
              setMediafirePassword('');
              fetchAccounts();
            } catch (err: any) {
              setMediafireError(err?.message || String(err));
            } finally {
              setMediafireLoading(false);
            }
          }}>
            <h3 className="modal-header">{t('connectMediaFire')}</h3>
            
            {mediafireError && (
              <div className="alert-panel" style={{ color: 'var(--md-sys-color-error)', borderColor: 'var(--md-sys-color-error)', marginBottom: '16px' }}>
                {mediafireError}
              </div>
            )}

            <div className="form-group">
              <label className="form-label">{t('mediafireEmailLabel')}</label>
              <input
                type="email"
                className="form-input"
                required
                placeholder="user@example.com"
                value={mediafireEmail}
                onChange={(e) => setMediafireEmail(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('mediafirePasswordLabel')}</label>
              <input
                type="password"
                className="form-input"
                required
                value={mediafirePassword}
                onChange={(e) => setMediafirePassword(e.target.value)}
              />
            </div>

            <div className="modal-footer">
              <button type="button" className="btn btn-text" disabled={mediafireLoading} onClick={() => setModal(null)}>{t('cancel')}</button>
              <button type="submit" className="btn btn-filled" disabled={mediafireLoading}>
                {mediafireLoading ? t('connecting') : t('create')}
              </button>
            </div>
          </form>
        </div>
      )}

      {modal?.type === 'fourshared' && (
        <div className="modal-overlay">
          <form className="modal-content" style={{ width: '440px' }} onSubmit={async (e) => {
            e.preventDefault();
            setFoursharedLoading(true);
            setFoursharedError('');
            try {
              // @ts-ignore
              await window.go?.main?.App?.AddFourSharedAccount(foursharedEmail, foursharedPassword);
              showToast(lang === 'id' ? 'Akun 4Shared berhasil terhubung!' : '4Shared account connected successfully!');
              setModal(null);
              setFoursharedEmail('');
              setFoursharedPassword('');
              fetchAccounts();
            } catch (err: any) {
              setFoursharedError(err?.message || String(err));
            } finally {
              setFoursharedLoading(false);
            }
          }}>
            <h3 className="modal-header">{t('connectFourShared')}</h3>
            
            {foursharedError && (
              <div className="alert-panel" style={{ color: 'var(--md-sys-color-error)', borderColor: 'var(--md-sys-color-error)', marginBottom: '16px' }}>
                {foursharedError}
              </div>
            )}

            <div className="form-group">
              <label className="form-label">{t('foursharedEmailLabel')}</label>
              <input
                type="email"
                className="form-input"
                required
                placeholder="user@example.com"
                value={foursharedEmail}
                onChange={(e) => setFoursharedEmail(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('foursharedPasswordLabel')}</label>
              <input
                type="password"
                className="form-input"
                required
                value={foursharedPassword}
                onChange={(e) => setFoursharedPassword(e.target.value)}
              />
            </div>

            <div className="modal-footer">
              <button type="button" className="btn btn-text" disabled={foursharedLoading} onClick={() => setModal(null)}>{t('cancel')}</button>
              <button type="submit" className="btn btn-filled" disabled={foursharedLoading}>
                {foursharedLoading ? t('connecting') : t('create')}
              </button>
            </div>
          </form>
        </div>
      )}

      {modal?.type === 'b2' && (
        <div className="modal-overlay">
          <form className="modal-content" style={{ width: '480px' }} onSubmit={async (e) => {
            e.preventDefault();
            setB2Loading(true);
            setB2Error('');
            try {
              // @ts-ignore
              await window.go?.main?.App?.AddB2Account(b2DisplayName || `B2 (${b2Bucket})`, b2KeyID, b2AppKey, b2Bucket);
              showToast(lang === 'id' ? 'Backblaze B2 berhasil terhubung!' : 'Backblaze B2 connected successfully!');
              setModal(null);
              fetchAccounts();
            } catch (err: any) {
              setB2Error(err?.message || String(err));
            } finally {
              setB2Loading(false);
            }
          }}>
            <h3 className="modal-header">{t('connectB2')}</h3>
            
            {b2Error && (
              <div className="alert-panel" style={{ color: 'var(--md-sys-color-error)', borderColor: 'var(--md-sys-color-error)', marginBottom: '16px' }}>
                {b2Error}
              </div>
            )}

            <div className="form-group">
              <label className="form-label">{t('displayNameLabel')}</label>
              <input
                type="text"
                className="form-input"
                placeholder="My B2 Storage"
                value={b2DisplayName}
                onChange={(e) => setB2DisplayName(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('b2KeyIDLabel')}</label>
              <input
                type="text"
                className="form-input"
                required
                value={b2KeyID}
                onChange={(e) => setB2KeyID(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('b2AppKeyLabel')}</label>
              <input
                type="password"
                className="form-input"
                required
                value={b2AppKey}
                onChange={(e) => setB2AppKey(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('b2BucketLabel')}</label>
              <input
                type="text"
                className="form-input"
                required
                placeholder="my-b2-bucket"
                value={b2Bucket}
                onChange={(e) => setB2Bucket(e.target.value)}
              />
            </div>

            <div className="modal-footer">
              <button type="button" className="btn btn-text" disabled={b2Loading} onClick={() => setModal(null)}>{t('cancel')}</button>
              <button type="submit" className="btn btn-filled" disabled={b2Loading}>
                {b2Loading ? t('connecting') : t('create')}
              </button>
            </div>
          </form>
        </div>
      )}

      {modal?.type === 'smb' && (
        <div className="modal-overlay">
          <form className="modal-content" style={{ width: '480px' }} onSubmit={async (e) => {
            e.preventDefault();
            setSmbLoading(true);
            setSmbError('');
            try {
              // @ts-ignore
              await window.go?.main?.App?.AddSMBAccount(smbDisplayName || `SMB (\\\\${smbHost}\\${smbShare})`, smbHost, smbShare, smbUsername, smbPassword);
              showToast(lang === 'id' ? 'Windows Share (SMB) berhasil terhubung!' : 'Windows Share (SMB) connected successfully!');
              setModal(null);
              fetchAccounts();
            } catch (err: any) {
              setSmbError(err?.message || String(err));
            } finally {
              setSmbLoading(false);
            }
          }}>
            <h3 className="modal-header">{t('connectSmb')}</h3>
            
            {smbError && (
              <div className="alert-panel" style={{ color: 'var(--md-sys-color-error)', borderColor: 'var(--md-sys-color-error)', marginBottom: '16px' }}>
                {smbError}
              </div>
            )}

            <div className="form-group">
              <label className="form-label">{t('displayNameLabel')}</label>
              <input
                type="text"
                className="form-input"
                placeholder="Office LAN Shared Folder"
                value={smbDisplayName}
                onChange={(e) => setSmbDisplayName(e.target.value)}
              />
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
              <div className="form-group">
                <label className="form-label">{t('smbHostLabel')}</label>
                <input
                  type="text"
                  className="form-input"
                  required
                  placeholder="192.168.1.100 or Hostname"
                  value={smbHost}
                  onChange={(e) => setSmbHost(e.target.value)}
                />
              </div>

              <div className="form-group">
                <label className="form-label">{t('smbShareLabel')}</label>
                <input
                  type="text"
                  className="form-input"
                  required
                  placeholder="SharedFolder"
                  value={smbShare}
                  onChange={(e) => setSmbShare(e.target.value)}
                />
              </div>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
              <div className="form-group">
                <label className="form-label">{t('smbUserLabel')}</label>
                <input
                  type="text"
                  className="form-input"
                  required
                  value={smbUsername}
                  onChange={(e) => setSmbUsername(e.target.value)}
                />
              </div>

              <div className="form-group">
                <label className="form-label">{t('smbPassLabel')}</label>
                <input
                  type="password"
                  className="form-input"
                  required
                  value={smbPassword}
                  onChange={(e) => setSmbPassword(e.target.value)}
                />
              </div>
            </div>

            <div className="modal-footer">
              <button type="button" className="btn btn-text" disabled={smbLoading} onClick={() => setModal(null)}>{t('cancel')}</button>
              <button type="submit" className="btn btn-filled" disabled={smbLoading}>
                {smbLoading ? t('connecting') : t('create')}
              </button>
            </div>
          </form>
        </div>
      )}

      {(modal?.type === 'ftp' || modal?.type === 'sftp') && (
        <div className="modal-overlay">
          <form className="modal-content" style={{ width: '480px' }} onSubmit={async (e) => {
            e.preventDefault();
            setServerLoading(true);
            setServerError('');
            try {
              if (modal?.type === 'ftp') {
                // @ts-ignore
                await window.go?.main?.App?.AddFTPAccount(serverDisplayName || `FTP (${serverHost})`, serverHost, Number(serverPort) || 21, serverUsername, serverPassword, serverBaseDir || '/');
                showToast(lang === 'id' ? 'Server FTP berhasil terhubung!' : 'FTP Server connected successfully!');
              } else {
                // @ts-ignore
                await window.go?.main?.App?.AddSFTPAccount(serverDisplayName || `SFTP (${serverHost})`, serverHost, Number(serverPort) || 22, serverUsername, serverPassword, serverBaseDir || '/');
                showToast(lang === 'id' ? 'Server SFTP berhasil terhubung!' : 'SFTP Server connected successfully!');
              }
              setModal(null);
              fetchAccounts();
            } catch (err: any) {
              setServerError(err?.message || String(err));
            } finally {
              setServerLoading(false);
            }
          }}>
            <h3 className="modal-header">{modal?.type === 'ftp' ? t('connectFtp') : t('connectSftp')}</h3>
            
            {serverError && (
              <div className="alert-panel" style={{ color: 'var(--md-sys-color-error)', borderColor: 'var(--md-sys-color-error)', marginBottom: '16px' }}>
                {serverError}
              </div>
            )}

            <div className="form-group">
              <label className="form-label">{t('displayNameLabel')}</label>
              <input
                type="text"
                className="form-input"
                placeholder={modal?.type === 'ftp' ? "My FTP Server" : "My VPS SFTP"}
                value={serverDisplayName}
                onChange={(e) => setServerDisplayName(e.target.value)}
              />
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: '12px' }}>
              <div className="form-group">
                <label className="form-label">{t('serverHostLabel')}</label>
                <input
                  type="text"
                  className="form-input"
                  required
                  placeholder="ftp.example.com or IP"
                  value={serverHost}
                  onChange={(e) => setServerHost(e.target.value)}
                />
              </div>

              <div className="form-group">
                <label className="form-label">{t('serverPortLabel')}</label>
                <input
                  type="number"
                  className="form-input"
                  required
                  value={serverPort}
                  onChange={(e) => setServerPort(Number(e.target.value))}
                />
              </div>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
              <div className="form-group">
                <label className="form-label">{t('serverUserLabel')}</label>
                <input
                  type="text"
                  className="form-input"
                  required
                  value={serverUsername}
                  onChange={(e) => setServerUsername(e.target.value)}
                />
              </div>

              <div className="form-group">
                <label className="form-label">{t('serverPassLabel')}</label>
                <input
                  type="password"
                  className="form-input"
                  required
                  value={serverPassword}
                  onChange={(e) => setServerPassword(e.target.value)}
                />
              </div>
            </div>

            <div className="form-group">
              <label className="form-label">{t('serverBaseDirLabel')}</label>
              <input
                type="text"
                className="form-input"
                placeholder="/"
                value={serverBaseDir}
                onChange={(e) => setServerBaseDir(e.target.value)}
              />
            </div>

            <div className="modal-footer">
              <button type="button" className="btn btn-text" disabled={serverLoading} onClick={() => setModal(null)}>{t('cancel')}</button>
              <button type="submit" className="btn btn-filled" disabled={serverLoading}>
                {serverLoading ? t('connecting') : t('create')}
              </button>
            </div>
          </form>
        </div>
      )}

      {modal?.type === 'webdav' && (
        <div className="modal-overlay">
          <form className="modal-content" style={{ width: '480px' }} onSubmit={handleWebdavSubmit}>
            <h3 className="modal-header">{t('connectWebdav')}</h3>
            
            {webdavError && (
              <div className="alert-panel" style={{ color: 'var(--md-sys-color-error)', borderColor: 'var(--md-sys-color-error)', marginBottom: '16px' }}>
                {webdavError}
              </div>
            )}

            <div className="form-group">
              <label className="form-label">{t('displayNameLabel')}</label>
              <input
                type="text"
                className="form-input"
                required
                placeholder="e.g. Nextcloud Storage"
                value={webdavName}
                onChange={(e) => setWebdavName(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('webdavUrlLabel')}</label>
              <input
                type="url"
                className="form-input"
                required
                placeholder={t('webdavUrlDesc')}
                value={webdavUrl}
                onChange={(e) => setWebdavUrl(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('webdavUserLabel')}</label>
              <input
                type="text"
                className="form-input"
                required
                value={webdavUsername}
                onChange={(e) => setWebdavUsername(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('webdavPassLabel')}</label>
              <input
                type="password"
                className="form-input"
                required
                value={webdavPassword}
                onChange={(e) => setWebdavPassword(e.target.value)}
              />
            </div>

            <div className="modal-footer">
              <button type="button" className="btn btn-text" disabled={webdavLoading} onClick={() => setModal(null)}>{t('cancel')}</button>
              <button type="submit" className="btn btn-filled" disabled={webdavLoading}>
                {webdavLoading ? t('connecting') : t('create')}
              </button>
            </div>
          </form>
        </div>
      )}

      {modal?.type === 's3' && (
        <div className="modal-overlay">
          <form className="modal-content" style={{ width: '480px' }} onSubmit={handleS3Submit}>
            <h3 className="modal-header">{t('connectS3')}</h3>
            
            {s3Error && (
              <div className="alert-panel" style={{ color: 'var(--md-sys-color-error)', borderColor: 'var(--md-sys-color-error)', marginBottom: '16px' }}>
                {s3Error}
              </div>
            )}

            <div className="form-group">
              <label className="form-label">{t('displayNameLabel')}</label>
              <input
                type="text"
                className="form-input"
                required
                placeholder="e.g. Cloudflare R2"
                value={s3Name}
                onChange={(e) => setS3Name(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('s3EndpointLabel')}</label>
              <input
                type="url"
                className="form-input"
                required
                placeholder="e.g. https://<id>.r2.cloudflarestorage.com"
                value={s3Endpoint}
                onChange={(e) => setS3Endpoint(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('s3BucketLabel')}</label>
              <input
                type="text"
                className="form-input"
                required
                placeholder="e.g. my-bucket"
                value={s3Bucket}
                onChange={(e) => setS3Bucket(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('s3AccessLabel')}</label>
              <input
                type="text"
                className="form-input"
                required
                value={s3AccessKey}
                onChange={(e) => setS3AccessKey(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('s3SecretLabel')}</label>
              <input
                type="password"
                className="form-input"
                required
                value={s3SecretKey}
                onChange={(e) => setS3SecretKey(e.target.value)}
              />
            </div>

            <div className="modal-footer">
              <button type="button" className="btn btn-text" disabled={s3Loading} onClick={() => setModal(null)}>{t('cancel')}</button>
              <button type="submit" className="btn btn-filled" disabled={s3Loading}>
                {s3Loading ? t('connecting') : t('create')}
              </button>
            </div>
          </form>
        </div>
      )}

      {modal?.type === 'telegram' && (
        <div className="modal-overlay">
          <form className="modal-content" style={{ width: '480px' }} onSubmit={handleTelegramSubmit}>
            <h3 className="modal-header">{t('connectTelegram')}</h3>
            
            {telegramError && (
              <div className="alert-panel" style={{ color: 'var(--md-sys-color-error)', borderColor: 'var(--md-sys-color-error)', marginBottom: '16px' }}>
                {telegramError}
              </div>
            )}

            <div className="form-group">
              <label className="form-label">{t('displayNameLabel')}</label>
              <input
                type="text"
                className="form-input"
                required
                placeholder="e.g. Telegram Bot Storage"
                value={telegramName}
                onChange={(e) => setTelegramName(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('telegramTokenLabel')}</label>
              <input
                type="text"
                className="form-input"
                required
                placeholder="e.g. 123456:ABC-DefGHI"
                value={telegramToken}
                onChange={(e) => setTelegramToken(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('telegramChatIdLabel')}</label>
              <input
                type="text"
                className="form-input"
                required
                placeholder="e.g. -100123456789"
                value={telegramChatID}
                onChange={(e) => setTelegramChatID(e.target.value)}
              />
            </div>

            <div className="modal-footer">
              <button type="button" className="btn btn-text" disabled={telegramLoading} onClick={() => setModal(null)}>{t('cancel')}</button>
              <button type="submit" className="btn btn-filled" disabled={telegramLoading}>
                {telegramLoading ? t('connecting') : t('create')}
              </button>
            </div>
          </form>
        </div>
      )}

      {modal?.type === 'telegram_user' && (
        <div className="modal-overlay">
          <div className="modal-content" style={{ width: '480px' }}>
            <h3 className="modal-header">{t('connectTelegramUser')}</h3>

            {tgUserError && (
              <div className="alert-panel" style={{ color: 'var(--md-sys-color-error)', borderColor: 'var(--md-sys-color-error)', marginBottom: '16px' }}>
                {tgUserError}
              </div>
            )}

            {tgUserStep === 'phone' && (
              <form onSubmit={handleTelegramUserPhoneSubmit}>
                <p style={{ fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)', marginBottom: '16px' }}>
                  {t('tgUserPhoneDesc')}
                </p>

                <div className="form-group">
                  <label className="form-label">{t('displayNameLabel')}</label>
                  <input
                    type="text"
                    className="form-input"
                    required
                    placeholder="e.g. My Saved Messages"
                    value={tgUserDisplayName}
                    onChange={(e) => setTgUserDisplayName(e.target.value)}
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">{t('telegramUserPhoneLabel')}</label>
                  <input
                    type="tel"
                    className="form-input"
                    required
                    placeholder="e.g. +62812345678"
                    value={tgUserPhone}
                    onChange={(e) => setTgUserPhone(e.target.value)}
                  />
                </div>

                <div className="modal-footer">
                  <button type="button" className="btn btn-text" disabled={tgUserLoading} onClick={() => setModal(null)}>{t('cancel')}</button>
                  <button type="submit" className="btn btn-filled" disabled={tgUserLoading}>
                    {tgUserLoading ? t('connecting') : t('next')}
                  </button>
                </div>
              </form>
            )}

            {(tgUserStep === 'code' || tgUserStep === 'password') && (
              <form onSubmit={handleTelegramUserCodeSubmit}>
                {tgUserStep === 'code' && (
                  <div className="form-group">
                    <label className="form-label">{t('telegramUserCodeLabel')}</label>
                    <input
                      type="text"
                      className="form-input"
                      required
                      value={tgUserCode}
                      onChange={(e) => setTgUserCode(e.target.value)}
                    />
                  </div>
                )}

                {tgUserStep === 'password' && (
                  <div className="form-group">
                    <label className="form-label">{t('telegramUserPassLabel')}</label>
                    <input
                      type="password"
                      className="form-input"
                      required
                      value={tgUserPassword}
                      onChange={(e) => setTgUserPassword(e.target.value)}
                    />
                  </div>
                )}

                <div className="modal-footer">
                  <button type="button" className="btn btn-text" disabled={tgUserLoading} onClick={() => setTgUserStep('phone')}>{t('cancel')}</button>
                  <button type="submit" className="btn btn-filled" disabled={tgUserLoading}>
                    {tgUserLoading ? t('connecting') : t('submit')}
                  </button>
                </div>
              </form>
            )}
          </div>
        </div>
      )}

      {modal?.type === 'credentials' && (
        <div className="modal-overlay">
          <form className="modal-content" style={{ width: '480px' }} onSubmit={handleCredentialsSubmit}>
            <h3 className="modal-header" style={{ textTransform: 'capitalize' }}>
              {modal.provider === 'telegram_user' ? t('telegramApiCreds') : `${modal.provider} ${t('oauthCredsHeader')}`}
            </h3>
            
            <div className="form-group">
              <label className="form-label">
                {modal.provider === 'telegram_user' 
                  ? t('telegramApiIdLabel') 
                  : (modal.provider === 'dropbox' 
                    ? 'App key' 
                    : (modal.provider === 'onedrive' 
                      ? 'Application (client) ID' 
                      : 'Client ID'))}
              </label>
              <input
                type="text"
                className="form-input"
                required
                value={credClientID}
                onChange={(e) => setCredClientID(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label className="form-label">
                {modal.provider === 'telegram_user' 
                  ? t('telegramApiHashLabel') 
                  : (modal.provider === 'dropbox' 
                    ? 'App secret' 
                    : (modal.provider === 'onedrive' 
                      ? 'Secret Value (Client Secret)' 
                      : 'Client Secret'))}
              </label>
              <input
                type="password"
                className="form-input"
                required
                value={credClientSecret}
                onChange={(e) => setCredClientSecret(e.target.value)}
              />
            </div>

            {modal.provider !== 'telegram_user' && (
              <div style={{
                marginTop: '12px',
                marginBottom: '12px',
                padding: '10px 14px',
                borderRadius: '8px',
                backgroundColor: 'var(--md-sys-color-surface-container-high)',
                border: '1px solid var(--md-sys-color-outline-variant)',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                gap: '12px'
              }}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '2px', minWidth: 0 }}>
                  <span style={{ fontSize: '10px', fontWeight: '600', color: 'var(--md-sys-color-primary)' }}>Redirect URI (Callback)</span>
                  <span style={{ fontSize: '12px', fontFamily: 'monospace', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>http://localhost:5998/oauth/callback</span>
                </div>
                <button
                  type="button"
                  className="btn btn-text"
                  style={{ fontSize: '11px', padding: '6px 12px', flexShrink: 0, border: '1px solid var(--md-sys-color-outline-variant)' }}
                  onClick={() => {
                    navigator.clipboard.writeText('http://localhost:5998/oauth/callback');
                    showToast('Copied redirect URI!');
                  }}
                >
                  {t('copyLink')}
                </button>
              </div>
            )}

            <p style={{ fontSize: '11px', color: 'var(--md-sys-color-on-surface-variant)', lineHeight: '1.4', margin: '4px 0 12px 0' }}>
              {modal.provider === 'telegram_user' ? t('telegramApiDesc') : t('oauthRedirectNote')}
            </p>

            <div className="modal-footer">
              <button type="button" className="btn btn-text" onClick={() => setModal(null)}>{t('cancel')}</button>
              <button type="submit" className="btn btn-filled">{t('saveCredentials')}</button>
            </div>
          </form>
        </div>
      )}

      {modal?.type === 'backup-task' && (
        <div className="modal-overlay">
          <form className="modal-content" style={{ width: '480px' }} onSubmit={handleAddSyncTask}>
            <h3 className="modal-header">
              {editingSyncTask 
                ? (lang === 'id' ? 'Edit Tugas Backup' : 'Edit Backup Task') 
                : t('addBackupTask')}
            </h3>

            <div className="form-group">
              <label className="form-label">{t('localFolder')}</label>
              <div style={{ display: 'flex', gap: '8px' }}>
                <input
                  type="text"
                  className="form-input"
                  required
                  readOnly
                  placeholder="No folder selected"
                  value={backupLocalPath}
                  style={{ flexGrow: 1 }}
                />
                {!editingSyncTask && (
                  <button
                    type="button"
                    className="btn btn-text"
                    style={{ border: '1px solid var(--md-sys-color-outline-variant)', flexShrink: 0 }}
                    onClick={handleSelectBackupFolder}
                  >
                    {t('selectFolder')}
                  </button>
                )}
              </div>
            </div>

            <div className="form-group">
              <label className="form-label">{t('destinationAccount')}</label>
              <select
                className="form-input"
                value={backupAccountID}
                onChange={(e) => setBackupAccountID(e.target.value)}
                style={{ height: '42px', padding: '0 8px' }}
              >
                <option value="auto">{t('autoAllocate')}</option>
                {accounts.filter(a => a.active).map(a => (
                  <option key={a.id} value={a.id}>{a.displayName} ({a.provider})</option>
                ))}
              </select>
            </div>

            <div className="form-group">
              <label className="form-label">{t('syncMode')}</label>
              <select
                className="form-input"
                value={backupSyncMode}
                onChange={(e) => setBackupSyncMode(e.target.value)}
                style={{ height: '42px', padding: '0 8px' }}
              >
                <option value="one-way">{t('oneWay')}</option>
                <option value="two-way">{t('twoWay')}</option>
              </select>
            </div>

            <div className="modal-footer">
              <button type="button" className="btn btn-text" onClick={() => setModal(null)}>{t('cancel')}</button>
              <button type="submit" className="btn btn-filled">
                {editingSyncTask 
                  ? (lang === 'id' ? 'Simpan Perubahan' : 'Save Changes') 
                  : t('addSyncTaskButton')}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Provider Setup Guide Modal */}
      {showGuideModal && activeGuide && PROVIDER_GUIDES[activeGuide] && (
        <div className="modal-overlay" style={{ zIndex: 3100 }}>
          <div className="modal-content" style={{ width: '560px', maxHeight: '85vh', overflowY: 'auto', borderRadius: '28px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
              <h3 className="modal-header" style={{ margin: 0 }}>{PROVIDER_GUIDES[activeGuide](t).title}</h3>
              <button className="icon-btn" onClick={() => setShowGuideModal(false)}>
                <IconClose />
              </button>
            </div>
            
            <div className="api-guide-box" style={{ padding: '0', border: 'none', background: 'transparent' }}>
              <div style={{ whiteSpace: 'pre-line' }}>
                {PROVIDER_GUIDES[activeGuide](t).steps}
              </div>
            </div>

            <div className="modal-footer" style={{ marginTop: '32px' }}>
              <button className="btn btn-filled" onClick={() => setShowGuideModal(false)}>
                {t('confirmLabel') || 'OK'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Action Dialog (Info, Confirm, Prompt) */}
      {actionDialog && (
        <div className="modal-overlay" style={{ zIndex: 3000 }}>
          <div className="modal-content" style={{ width: '312px', padding: '24px', borderRadius: '28px', textAlign: 'center' }}>
            <div className="dialog-icon">
              {actionDialog.variant === 'danger' && <span style={{ color: 'var(--md-sys-color-error)' }}><IconDelete /></span>}
              {actionDialog.variant === 'warning' && <span style={{ color: 'var(--md-sys-color-tertiary)' }}><IconWarning /></span>}
              {(actionDialog.variant === 'info' || !actionDialog.variant) && <span style={{ color: 'var(--md-sys-color-primary)' }}><IconInfo /></span>}
            </div>

            <h3 className="modal-header" style={{ marginBottom: '16px', fontSize: '24px', fontWeight: '400' }}>
              {actionDialog.title}
            </h3>
            
            <p style={{ fontSize: '14px', lineHeight: '1.5', color: 'var(--md-sys-color-on-surface-variant)', marginBottom: '24px', wordBreak: 'break-all', overflowWrap: 'anywhere' }}>
              {actionDialog.message}
            </p>

            {actionDialog.type === 'prompt' && (
              <div className="form-group" style={{ marginBottom: '24px', textAlign: 'left' }}>
                <label className="form-label">{actionDialog.inputLabel}</label>
                <input
                  type="text"
                  className="form-input"
                  autoFocus
                  value={dialogInput}
                  onChange={(e) => setDialogInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      actionDialog.onConfirm(dialogInput);
                      closeActionDialog();
                    }
                  }}
                />
              </div>
            )}

            <div className="modal-footer" style={{ justifyContent: 'flex-end', gap: '8px', padding: 0, marginTop: '24px' }}>
              {(actionDialog.type === 'confirm' || actionDialog.type === 'prompt') && (
                <button 
                  className="btn btn-text" 
                  onClick={closeActionDialog}
                  style={{ padding: '10px 12px' }}
                >
                  {actionDialog.type === 'confirm' ? (actionDialog.cancelLabel || t('cancel')) : (actionDialog.cancelLabel || t('cancel'))}
                </button>
              )}
              <button 
                className={`btn ${actionDialog.variant === 'danger' ? 'btn-filled-error' : 'btn-filled'}`}
                style={{ 
                  padding: '10px 12px',
                  ...(actionDialog.variant === 'danger' ? { backgroundColor: 'var(--md-sys-color-error)', color: 'var(--md-sys-color-on-error)' } : {})
                }}
                onClick={async () => {
                  if (actionDialog.type === 'confirm') {
                    await actionDialog.onConfirm();
                  } else if (actionDialog.type === 'prompt') {
                    await actionDialog.onConfirm(dialogInput);
                  }
                  closeActionDialog();
                }}
              >
                {actionDialog.type === 'info' ? (actionDialog.confirmLabel || 'OK') : (actionDialog.confirmLabel || (actionDialog.type === 'prompt' ? t('submit') : t('yes')))}
              </button>
            </div>
          </div>
        </div>
      )}

      {showUpdateModal && updateInfo && (
        <div style={{
          position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
          backgroundColor: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center',
          justifyContent: 'center', zIndex: 9999, animation: 'fadeIn 0.25s ease',
          backdropFilter: 'blur(4px)'
        }}>
          <div style={{
            backgroundColor: 'var(--md-sys-color-surface-container-high, #252b36)', width: '420px', padding: '28px',
            borderRadius: '28px', textAlign: 'left', boxShadow: '0 8px 30px rgba(0,0,0,0.3)',
            border: '1px solid var(--md-sys-color-outline-variant, #394252)',
            fontFamily: 'Inter, system-ui, sans-serif',
            display: 'flex', flexDirection: 'column', gap: '16px'
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <div style={{
                background: 'rgba(59, 130, 246, 0.2)',
                borderRadius: '12px', padding: '8px', display: 'flex', alignItems: 'center', justifyContent: 'center'
              }}>
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="var(--md-sys-color-primary, #3b82f6)" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                  <polyline points="7 10 12 15 17 10" />
                  <line x1="12" y1="15" x2="12" y2="3" />
                </svg>
              </div>
              <h3 style={{ margin: 0, fontSize: '20px', fontWeight: '500', color: 'var(--md-sys-color-on-surface, #fff)' }}>
                {lang === 'id' ? 'Pembaruan Tersedia' : 'Update Available'}
              </h3>
            </div>

            <p style={{ fontSize: '14px', lineHeight: '1.6', color: 'var(--md-sys-color-on-surface-variant, #a0aec0)', margin: 0 }}>
              {lang === 'id' 
                ? `Versi baru (${updateInfo.latest_version}) telah dirilis. Versi Anda saat ini adalah ${appVersion}.`
                : `A new version (${updateInfo.latest_version}) is available. Your current version is ${appVersion}.`}
            </p>

            {updateInfo.release_notes && (
              <div style={{
                backgroundColor: 'var(--md-sys-color-surface-container-low, #1e232d)',
                borderRadius: '16px', padding: '12px 16px', fontSize: '13px',
                color: 'var(--md-sys-color-on-surface-variant, #a0aec0)', maxHeight: '160px',
                overflowY: 'auto', border: '1px solid var(--md-sys-color-outline-variant, #394252)',
                whiteSpace: 'pre-wrap', lineHeight: '1.5'
              }}>
                <strong style={{ display: 'block', marginBottom: '6px', color: 'var(--md-sys-color-on-surface, #fff)' }}>
                  {lang === 'id' ? 'Catatan Rilis:' : 'Release Notes:'}
                </strong>
                {updateInfo.release_notes}
              </div>
            )}

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', marginTop: '8px' }}>
              <button
                onClick={() => setShowUpdateModal(false)}
                style={{
                  background: 'transparent', border: 'none', color: 'var(--md-sys-color-primary, #3b82f6)',
                  padding: '10px 20px', borderRadius: '100px', cursor: 'pointer', fontWeight: 500,
                  fontSize: '14px', transition: 'background-color 0.2s'
                }}
                onMouseEnter={e => e.currentTarget.style.backgroundColor = 'rgba(59, 130, 246, 0.1)'}
                onMouseLeave={e => e.currentTarget.style.backgroundColor = 'transparent'}
              >
                {lang === 'id' ? 'Nanti Saja' : 'Later'}
              </button>
              <button
                onClick={() => {
                  OpenReleaseURL(updateInfo.update_url);
                  setShowUpdateModal(false);
                }}
                style={{
                  background: 'var(--md-sys-color-primary, #3b82f6)', border: 'none', color: '#fff',
                  padding: '10px 20px', borderRadius: '100px', cursor: 'pointer', fontWeight: 500,
                  fontSize: '14px', display: 'flex', alignItems: 'center', gap: '8px',
                  boxShadow: '0 1px 3px rgba(0,0,0,0.2)', transition: 'filter 0.2s'
                }}
                onMouseEnter={e => e.currentTarget.style.filter = 'brightness(1.1)'}
                onMouseLeave={e => e.currentTarget.style.filter = 'none'}
              >
                {lang === 'id' ? 'Perbarui Sekarang' : 'Update Now'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Create Web Share Modal */}
      {createShareFile && (
        <div style={{
          position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
          backgroundColor: 'rgba(0, 0, 0, 0.6)', backdropFilter: 'blur(4px)',
          display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 3000
        }}>
          <div style={{
            backgroundColor: 'var(--md-sys-color-surface-container-high, #1e293b)',
            border: '1px solid var(--md-sys-color-outline-variant, rgba(255,255,255,0.1))',
            borderRadius: '24px', padding: '28px', width: '100%', maxWidth: '420px',
            boxShadow: 'var(--shadow-3, 0 16px 48px rgba(0,0,0,0.5))', color: 'var(--md-sys-color-on-surface, #fff)'
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '16px' }}>
              <div style={{
                width: '40px', height: '40px', borderRadius: '50%', background: 'rgba(56,189,248,0.15)',
                display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#38bdf8'
              }}>
                <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
              </div>
              <div>
                <h3 style={{ margin: 0, fontSize: '18px', fontWeight: '700' }}>
                  {lang === 'id' ? 'Berbagi Web (Lokal/Public)' : 'Create Web Share'}
                </h3>
                <p style={{ margin: 0, fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant, #94a3b8)' }}>
                  {createShareFile.name}
                </p>
              </div>
            </div>

            <form onSubmit={handleCreateWebShareSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <div>
                <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '6px', color: 'var(--md-sys-color-on-surface-variant, #94a3b8)' }}>
                  {lang === 'id' ? 'Password Proteksi (Opsional)' : 'Protection Password (Optional)'}
                </label>
                <input
                  type="password"
                  value={sharePassword}
                  onChange={e => setSharePassword(e.target.value)}
                  placeholder={lang === 'id' ? 'Kosongkan jika tanpa password' : 'Leave empty for no password'}
                  style={{
                    width: '100%', padding: '12px 16px', borderRadius: '12px',
                    border: '1px solid var(--md-sys-color-outline-variant, rgba(255,255,255,0.15))',
                    backgroundColor: 'rgba(0,0,0,0.3)', color: '#fff', fontSize: '14px',
                    boxSizing: 'border-box', outline: 'none'
                  }}
                />
              </div>

              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '8px' }}>
                <button
                  type="button"
                  onClick={() => setCreateShareFile(null)}
                  style={{
                    padding: '10px 20px', borderRadius: '100px', border: '1px solid var(--md-sys-color-outline-variant, rgba(255,255,255,0.2))',
                    backgroundColor: 'transparent', color: '#fff', fontSize: '13px', cursor: 'pointer'
                  }}
                >
                  {lang === 'id' ? 'Batal' : 'Cancel'}
                </button>
                <button
                  type="submit"
                  disabled={creatingShare}
                  style={{
                    padding: '10px 24px', borderRadius: '100px', border: 'none',
                    backgroundColor: '#38bdf8', color: '#0f172a', fontSize: '13px', fontWeight: '700', cursor: 'pointer'
                  }}
                >
                  {creatingShare ? (lang === 'id' ? 'Membuat...' : 'Creating...') : (lang === 'id' ? 'Buat Link' : 'Create Link')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {toast.visible && (
        <div className="toast-notification" style={{
          position: 'fixed',
          bottom: '24px',
          left: '50%',
          transform: 'translateX(-50%)',
          backgroundColor: 'var(--md-sys-color-inverse-surface)',
          color: 'var(--md-sys-color-inverse-on-surface)',
          padding: '12px 24px',
          borderRadius: '100px',
          boxShadow: 'var(--shadow-3)',
          zIndex: 5000,
          fontSize: '13px',
          fontWeight: '500',
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
          pointerEvents: 'none'
        }}>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" style={{ color: 'var(--md-sys-color-primary-container)' }}><path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/></svg>
          <span>{toast.message}</span>
        </div>
      )}
    </div>
  );
}

export default App;
