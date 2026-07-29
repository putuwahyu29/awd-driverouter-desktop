import React, { useState, useEffect } from 'react';
import {
  // @ts-ignore
  GetVirtualDriveStatus,
  // @ts-ignore
  MountVirtualDrive,
  // @ts-ignore
  UnmountVirtualDrive,
  // @ts-ignore
  StartNativeWebDAVServer,
  // @ts-ignore
  StopNativeWebDAVServer,
  // @ts-ignore
  GetMountedVirtualDrives,
  // @ts-ignore
  GetAccounts,
  // @ts-ignore
  SetAutoMountOnStartup
} from '../../wailsjs/go/main/App';
import { TRANSLATIONS } from '../translations';

interface VirtualDriveStatus {
  activeDriveCount: number;
  availableDrives: string[];
  webdavServerPort: number;
  webdavRunning: boolean;
  webdavPassword?: string;
  autoMountOnStart?: boolean;
  autoMountLetter?: string;
}

interface MountedDriveInfo {
  id: string;
  driveLetter: string;
  accountId: string;
  targetName: string;
  url: string;
  startTime: string;
  status: string;
}

interface AccountRecord {
  id: string;
  provider: string;
  displayName: string;
  email?: string;
  active: boolean;
}

interface VirtualDriveViewProps {
  theme?: 'light' | 'dark';
  lang?: 'en' | 'id';
}

const generateRandomPIN = () => Math.floor(100000 + Math.random() * 900000).toString();

export default function VirtualDriveView({ theme = 'dark', lang = 'en' }: VirtualDriveViewProps) {
  const [status, setStatus] = useState<VirtualDriveStatus | null>(null);
  const [mountedDrives, setMountedDrives] = useState<MountedDriveInfo[]>([]);
  const [accounts, setAccounts] = useState<AccountRecord[]>([]);
  const [loading, setLoading] = useState<boolean>(true);

  // Form states
  const [selectedAccountID, setSelectedAccountID] = useState<string>('all');
  const [selectedDrive, setSelectedDrive] = useState<string>('Z:');
  const [webdavPort, setWebdavPort] = useState<number>(8085);
  const [webdavPassword, setWebdavPassword] = useState<string>(generateRandomPIN);
  const [webdavURL, setWebdavURL] = useState<string>('');
  const [autoMount, setAutoMount] = useState<boolean>(false);
  const [actionMsg, setActionMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  // Animated Loading States for buttons
  const [isMounting, setIsMounting] = useState<boolean>(false);
  const [isStartingWebDAV, setIsStartingWebDAV] = useState<boolean>(false);

  // Confirmation Modal state for dangerous action (unmounting drive)
  const [confirmUnmountDriveLetter, setConfirmUnmountDriveLetter] = useState<string | null>(null);

  const dict = TRANSLATIONS[lang] || TRANSLATIONS.en;
  const t = (key: string) => dict[key] || key;

  const refreshData = async () => {
    setLoading(true);
    try {
      const st = await GetVirtualDriveStatus();
      setStatus(st);
      if (st.webdavPassword) {
        setWebdavPassword(st.webdavPassword);
      }
      if (st.autoMountOnStart !== undefined) {
        setAutoMount(st.autoMountOnStart);
      }
      if (st.autoMountLetter) {
        setSelectedDrive(st.autoMountLetter);
      }

      const mounted = await GetMountedVirtualDrives();
      setMountedDrives(mounted || []);

      const accs = await GetAccounts();
      setAccounts(accs || []);
    } catch (e: any) {
      console.error('Error checking Virtual Drive status:', e);
    } finally {
      setLoading(false);
    }
  };

  const handleToggleAutoMount = async (checked: boolean) => {
    setAutoMount(checked);
    try {
      await SetAutoMountOnStartup(checked, selectedDrive);
      setActionMsg({
        type: 'success',
        text: checked
          ? (lang === 'id' ? 'Auto-Mount saat Startup diaktifkan.' : 'Auto-Mount on Startup enabled.')
          : (lang === 'id' ? 'Auto-Mount saat Startup dimatikan.' : 'Auto-Mount on Startup disabled.')
      });
    } catch (err: any) {
      console.error('Error setting AutoMount:', err);
    }
  };

  useEffect(() => {
    refreshData();
    const interval = setInterval(() => {
      refreshData();
    }, 4000);
    return () => clearInterval(interval);
  }, []);

  const handleMountDrive = async () => {
    setIsMounting(true);
    setActionMsg(null);
    try {
      const result = await MountVirtualDrive(selectedAccountID, selectedDrive);
      setActionMsg({
        type: 'success',
        text: lang === 'id'
          ? `Sukses menghubungkan Drive ${result.driveLetter} (127.0.0.1) ke Windows Explorer!`
          : `Successfully connected Drive ${result.driveLetter} (127.0.0.1) to Windows Explorer!`
      });
      await refreshData();
    } catch (err: any) {
      setActionMsg({
        type: 'error',
        text: lang === 'id'
          ? `Gagal menghubungkan Drive: ${err?.message || err}`
          : `Connection failed: ${err?.message || err}`
      });
    } finally {
      setIsMounting(false);
    }
  };

  const executeUnmountDrive = async (driveLetter: string) => {
    try {
      setActionMsg(null);
      await UnmountVirtualDrive(driveLetter);
      setActionMsg({
        type: 'success',
        text: lang === 'id'
          ? `Drive ${driveLetter} berhasil dilepaskan.`
          : `Drive ${driveLetter} disconnected successfully.`
      });
      setConfirmUnmountDriveLetter(null);
      refreshData();
    } catch (err: any) {
      setActionMsg({
        type: 'error',
        text: lang === 'id'
          ? `Gagal melepaskan drive: ${err?.message || err}`
          : `Disconnect failed: ${err?.message || err}`
      });
      setConfirmUnmountDriveLetter(null);
    }
  };

  const handleToggleWebDAV = async () => {
    setIsStartingWebDAV(true);
    setActionMsg(null);
    try {
      if (status?.webdavRunning) {
        // Stop WebDAV Server
        await StopNativeWebDAVServer();
        setWebdavURL('');
        setActionMsg({
          type: 'success',
          text: lang === 'id'
            ? 'Akses Wi-Fi berhasil dimatikan.'
            : 'Wi-Fi Access stopped successfully.'
        });
      } else {
        // Start WebDAV Server with Password
        const activePass = webdavPassword || generateRandomPIN();
        setWebdavPassword(activePass);
        const url = await StartNativeWebDAVServer(webdavPort, activePass);
        setWebdavURL(url);
        setActionMsg({
          type: 'success',
          text: lang === 'id'
            ? `Akses Wi-Fi aktif di ${url}`
            : `Wi-Fi Access active at ${url}`
        });
      }
      await refreshData();
    } catch (err: any) {
      setActionMsg({
        type: 'error',
        text: lang === 'id'
          ? `Gagal mengubah status Akses Wi-Fi: ${err?.message || err}`
          : `Failed to update Wi-Fi access: ${err?.message || err}`
      });
    } finally {
      setIsStartingWebDAV(false);
    }
  };

  return (
    <div style={{ background: 'transparent', width: '100%' }}>
      {/* Header Banner */}
      <div
        className="dashboard-card"
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '20px 24px',
          margin: '0 0 24px 0'
        }}
      >
        <div>
          <h2 style={{ margin: 0, fontSize: '18px', fontWeight: 600 }}>{t('virtualDriveTitle')}</h2>
          <p style={{ margin: '4px 0 0', fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)' }}>
            {t('virtualDriveDesc')}
          </p>
        </div>

        {/* Status Pill & Refresh */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <button
            type="button"
            className="btn btn-outlined"
            onClick={refreshData}
            style={{ fontSize: '12px', padding: '6px 14px' }}
          >
            {t('virtualDriveRefreshBtn')}
          </button>
          <div
            style={{
              padding: '6px 14px',
              borderRadius: '20px',
              fontSize: '12px',
              fontWeight: 600,
              background: 'rgba(34, 197, 94, 0.15)',
              color: '#22c55e',
              border: '1px solid rgba(34, 197, 94, 0.3)'
            }}
          >
            Drive Engine Ready
          </div>
        </div>
      </div>

      {/* Action Notification with High-Contrast Red for Errors */}
      {actionMsg && (
        <div
          style={{
            padding: '12px 16px',
            borderRadius: '10px',
            marginBottom: '20px',
            fontSize: '13px',
            fontWeight: 600,
            background: actionMsg.type === 'success' ? 'rgba(34, 197, 94, 0.15)' : 'rgba(239, 68, 68, 0.25)',
            color: actionMsg.type === 'success' ? '#22c55e' : '#f87171',
            border: actionMsg.type === 'success' ? '1px solid rgba(34, 197, 94, 0.3)' : '1px solid #f87171'
          }}
        >
          {actionMsg.text}
        </div>
      )}

      {/* 1. CONNECT DRIVE SECTION */}
      <div className="dashboard-card" style={{ padding: '24px', margin: '0 0 24px 0' }}>
        <h3 style={{ margin: '0 0 6px 0', fontSize: '16px', fontWeight: 600 }}>{t('virtualDriveMountBtn')}</h3>
        <p style={{ margin: '0 0 16px 0', fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)' }}>
          {t('virtualDriveDesc')}
        </p>

        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '16px', alignItems: 'flex-end' }}>
          <div style={{ flex: '1 1 220px' }}>
            <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, marginBottom: '6px', color: 'var(--md-sys-color-on-surface-variant)' }}>
              {t('virtualDriveTargetAccount')}
            </label>
            <select
              className="form-input"
              value={selectedAccountID}
              onChange={e => setSelectedAccountID(e.target.value)}
              disabled={isMounting}
              style={{ width: '100%', height: '42px', padding: '0 12px', borderRadius: '8px' }}
            >
              <option value="all">{t('virtualDriveAllAccounts')}</option>
              {accounts.filter(a => a.active).map(acc => (
                <option key={acc.id} value={acc.id}>
                  {acc.displayName} ({acc.provider.toUpperCase()})
                </option>
              ))}
            </select>
          </div>

          <div style={{ flex: '0 0 120px' }}>
            <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, marginBottom: '6px', color: 'var(--md-sys-color-on-surface-variant)' }}>
              {t('virtualDriveLetter')}
            </label>
            <select
              className="form-input"
              value={selectedDrive}
              onChange={e => setSelectedDrive(e.target.value)}
              disabled={isMounting}
              style={{ width: '100%', height: '42px', padding: '0 12px', borderRadius: '8px' }}
            >
              {(status?.availableDrives || ['Z:', 'Y:', 'X:', 'W:']).map(letter => (
                <option key={letter} value={letter}>
                  Drive {letter}
                </option>
              ))}
            </select>
          </div>

          <button
            type="button"
            className="btn btn-filled"
            onClick={handleMountDrive}
            disabled={isMounting}
            style={{
              height: '42px',
              padding: '0 24px',
              fontWeight: 600,
              display: 'inline-flex',
              alignItems: 'center',
              gap: '8px',
              opacity: isMounting ? 0.75 : 1,
              cursor: isMounting ? 'not-allowed' : 'pointer'
            }}
          >
            {isMounting && (
              <span
                style={{
                  width: '14px',
                  height: '14px',
                  border: '2px solid rgba(255,255,255,0.3)',
                  borderTop: '2px solid #ffffff',
                  borderRadius: '50%',
                  animation: 'spin 1s linear infinite'
                }}
              />
            )}
            {isMounting
              ? (lang === 'id' ? 'Menghubungkan...' : 'Connecting...')
              : t('virtualDriveMountBtn')}
          </button>
        </div>

        {/* Auto-Mount on Startup Toggle */}
        <div style={{ marginTop: '20px', paddingTop: '16px', borderTop: '1px dashed var(--md-sys-color-outline-variant)', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '16px' }}>
          <div>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: 'var(--md-sys-color-on-surface)' }}>
              {t('virtualDriveAutoMountLabel')}
            </label>
            <p style={{ margin: '2px 0 0 0', fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)' }}>
              {t('virtualDriveAutoMountDesc')}
            </p>
          </div>
          <label className="switch" style={{ flexShrink: 0 }}>
            <input
              type="checkbox"
              checked={autoMount}
              onChange={e => handleToggleAutoMount(e.target.checked)}
            />
            <span className="slider round"></span>
          </label>
        </div>

        {/* Active Mounts List */}
        <div style={{ marginTop: '24px' }}>
          <h4 style={{ margin: '0 0 12px 0', fontSize: '14px', fontWeight: 600 }}>{t('virtualDriveActiveHeader')}</h4>
          {mountedDrives.length === 0 ? (
            <p style={{ fontSize: '13px', color: 'var(--md-sys-color-on-surface-variant)', margin: 0 }}>{t('virtualDriveNoActive')}</p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
              {mountedDrives.map(m => (
                <div
                  key={m.id}
                  style={{
                    padding: '14px 18px',
                    borderRadius: '10px',
                    background: 'var(--md-sys-color-surface-container)',
                    border: '1px solid var(--md-sys-color-outline-variant)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between'
                  }}
                >
                  <div>
                    <span style={{ fontWeight: 700, fontSize: '16px', color: 'var(--md-sys-color-primary)', marginRight: '12px' }}>{m.driveLetter}</span>
                    <span style={{ fontWeight: 600, fontSize: '14px' }}>{m.targetName}</span>
                    <span style={{ marginLeft: '12px', fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)' }}>({m.url})</span>
                  </div>

                  {/* High Contrast Red Button for Unmounting Drive */}
                  <button
                    type="button"
                    onClick={() => setConfirmUnmountDriveLetter(m.driveLetter)}
                    style={{
                      borderRadius: '8px',
                      border: '1px solid #f87171',
                      background: 'rgba(239, 68, 68, 0.15)',
                      color: '#f87171',
                      fontWeight: 600,
                      fontSize: '13px',
                      padding: '6px 16px',
                      cursor: 'pointer',
                      transition: 'all 0.2s'
                    }}
                  >
                    {t('virtualDriveUnmountBtn')}
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* 2. WI-FI FILE ACCESS SECTION */}
      <div className="dashboard-card" style={{ padding: '24px', margin: 0 }}>
        <h3 style={{ margin: '0 0 6px 0', fontSize: '16px', fontWeight: 600 }}>{t('virtualDriveWebDAVHeader')}</h3>
        <p style={{ margin: '0 0 16px 0', fontSize: '12px', color: 'var(--md-sys-color-on-surface-variant)' }}>
          {t('virtualDriveWebDAVDesc')}
        </p>

        <div style={{ display: 'flex', gap: '16px', alignItems: 'flex-end', flexWrap: 'wrap' }}>
          <div style={{ flex: '0 0 110px' }}>
            <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, marginBottom: '6px', color: 'var(--md-sys-color-on-surface-variant)' }}>
              Port Wi-Fi
            </label>
            <input
              type="number"
              className="form-input"
              value={webdavPort}
              onChange={e => setWebdavPort(parseInt(e.target.value) || 8085)}
              disabled={isStartingWebDAV || status?.webdavRunning}
              style={{ width: '100%', height: '42px', padding: '0 12px', borderRadius: '8px' }}
            />
          </div>

          <div style={{ flex: '1 1 240px' }}>
            <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, marginBottom: '6px', color: 'var(--md-sys-color-on-surface-variant)' }}>
              {lang === 'id' ? 'Password / PIN Wi-Fi (Acak Default)' : 'Wi-Fi Password / PIN (Random Default)'}
            </label>
            <div style={{ display: 'flex', gap: '8px' }}>
              <input
                type="text"
                className="form-input"
                value={webdavPassword}
                onChange={e => setWebdavPassword(e.target.value)}
                disabled={isStartingWebDAV || status?.webdavRunning}
                style={{ width: '100%', height: '42px', padding: '0 12px', borderRadius: '8px', fontWeight: 700, letterSpacing: '1px' }}
              />
              {!status?.webdavRunning && (
                <button
                  type="button"
                  className="btn btn-outlined"
                  onClick={() => setWebdavPassword(generateRandomPIN())}
                  disabled={isStartingWebDAV}
                  style={{ height: '42px', padding: '0 14px', fontSize: '12px', whiteSpace: 'nowrap' }}
                >
                  {lang === 'id' ? 'Acak PIN' : 'Random PIN'}
                </button>
              )}
            </div>
          </div>

          <button
            type="button"
            className={status?.webdavRunning ? "btn" : "btn btn-filled"}
            onClick={handleToggleWebDAV}
            disabled={isStartingWebDAV}
            style={{
              height: '42px',
              padding: '0 24px',
              fontWeight: 600,
              display: 'inline-flex',
              alignItems: 'center',
              gap: '8px',
              borderRadius: '8px',
              border: status?.webdavRunning ? '1px solid #f87171' : undefined,
              background: status?.webdavRunning ? 'rgba(239, 68, 68, 0.2)' : undefined,
              color: status?.webdavRunning ? '#f87171' : undefined,
              opacity: isStartingWebDAV ? 0.75 : 1,
              cursor: isStartingWebDAV ? 'not-allowed' : 'pointer',
              transition: 'all 0.2s',
              boxShadow: status?.webdavRunning ? '0 2px 10px rgba(239, 68, 68, 0.25)' : undefined
            }}
          >
            {isStartingWebDAV && (
              <span
                style={{
                  width: '14px',
                  height: '14px',
                  border: status?.webdavRunning ? '2px solid rgba(248, 113, 113, 0.3)' : '2px solid rgba(255,255,255,0.3)',
                  borderTop: status?.webdavRunning ? '2px solid #f87171' : '2px solid #ffffff',
                  borderRadius: '50%',
                  animation: 'spin 1s linear infinite'
                }}
              />
            )}
            {isStartingWebDAV
              ? (status?.webdavRunning ? (lang === 'id' ? 'Mematikan...' : 'Stopping...') : (lang === 'id' ? 'Mengaktifkan...' : 'Starting...'))
              : (status?.webdavRunning ? t('virtualDriveStopWebDAVBtn') : t('virtualDriveStartWebDAVBtn'))}
          </button>
        </div>

        {/* Clear Explanation Box when Active */}
        {status?.webdavRunning && (
          <div
            style={{
              marginTop: '20px',
              padding: '16px',
              borderRadius: '12px',
              background: 'rgba(34, 197, 94, 0.12)',
              border: '1px solid rgba(34, 197, 94, 0.3)',
              color: 'var(--md-sys-color-on-surface)'
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px', flexWrap: 'wrap' }}>
              <span style={{ width: '10px', height: '10px', borderRadius: '50%', background: '#22c55e' }} />
              <strong style={{ fontSize: '14px', color: '#22c55e' }}>{t('virtualDriveWebDAVActive')}</strong>
              <code style={{ fontSize: '14px', fontWeight: 700, padding: '2px 8px', borderRadius: '6px', background: 'rgba(34, 197, 94, 0.2)', color: '#22c55e' }}>
                {webdavURL || `http://127.0.0.1:${status.webdavServerPort}/`}
              </code>
              <span style={{ fontSize: '13px', color: '#eab308', fontWeight: 700, background: 'rgba(234, 179, 8, 0.15)', padding: '4px 10px', borderRadius: '6px', border: '1px solid rgba(234, 179, 8, 0.3)' }}>
                PIN Akses: {webdavPassword || status.webdavPassword}
              </span>
            </div>
            <p style={{ margin: 0, fontSize: '12px', lineHeight: '1.5', color: 'var(--md-sys-color-on-surface-variant)' }}>
              {t('virtualDriveWebDAVActiveExplanation')}
            </p>
          </div>
        )}
      </div>

      {/* CONFIRMATION MODAL FOR DANGEROUS ACTION (UNMOUNT DRIVE) */}
      {confirmUnmountDriveLetter && (
        <div
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: 'rgba(0, 0, 0, 0.65)',
            backdropFilter: 'blur(4px)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 9999,
            padding: '20px'
          }}
        >
          <div
            style={{
              width: '100%',
              maxWidth: '440px',
              borderRadius: '16px',
              background: 'var(--md-sys-color-surface, #1e293b)',
              color: 'var(--md-sys-color-on-surface, #f8fafc)',
              border: '1px solid var(--md-sys-color-outline-variant, #334155)',
              boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.4)',
              padding: '24px'
            }}
          >
            <h3 style={{ margin: '0 0 10px 0', fontSize: '18px', fontWeight: 700, color: '#f87171' }}>
              {lang === 'id' ? 'Konfirmasi Melepaskan Drive' : 'Confirm Disconnect Drive'}
            </h3>
            <p style={{ margin: '0 0 24px 0', fontSize: '13px', lineHeight: '1.5', color: 'var(--md-sys-color-on-surface-variant)' }}>
              {lang === 'id'
                ? `Apakah Anda yakin ingin melepaskan Drive ${confirmUnmountDriveLetter}? Akses file lokal ke folder ini pada Windows File Explorer akan terputus.`
                : `Are you sure you want to disconnect Drive ${confirmUnmountDriveLetter}? Local file access to this cloud drive in Windows Explorer will be disconnected.`}
            </p>

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px' }}>
              <button
                type="button"
                className="btn btn-outlined"
                onClick={() => setConfirmUnmountDriveLetter(null)}
                style={{ padding: '8px 20px', borderRadius: '8px', fontSize: '13px', fontWeight: 600 }}
              >
                {lang === 'id' ? 'Batal' : 'Cancel'}
              </button>
              <button
                type="button"
                onClick={() => executeUnmountDrive(confirmUnmountDriveLetter)}
                style={{
                  padding: '8px 20px',
                  borderRadius: '8px',
                  border: 'none',
                  background: '#dc2626',
                  color: '#ffffff',
                  fontSize: '13px',
                  fontWeight: 600,
                  cursor: 'pointer',
                  boxShadow: '0 2px 8px rgba(220, 38, 38, 0.4)'
                }}
              >
                {lang === 'id' ? 'Ya, Lepaskan Drive' : 'Yes, Disconnect Drive'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
