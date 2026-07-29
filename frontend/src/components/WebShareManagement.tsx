import React, { useState, useEffect } from 'react';
import {
  GetWebShares,
  DeleteWebShare,
  UpdateWebSharePassword,
  TogglePublicTunnel,
  GetTunnelPublicUrl,
  IsTunnelRunning,
  GetLocalIPAddress,
  GetWebServerPort
} from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { WebShareItem } from '../types';

interface WebShareManagementProps {
  lang: 'en' | 'id';
  addToast?: (message: string, type?: 'info' | 'error' | 'success') => void;
}

export default function WebShareManagement({ lang, addToast }: WebShareManagementProps) {
  const [shares, setShares] = useState<WebShareItem[]>([]);
  const [localIp, setLocalIp] = useState<string>('127.0.0.1');
  const [port, setPort] = useState<number>(0);
  const [publicUrl, setPublicUrl] = useState<string>('');
  const [isTunneling, setIsTunneling] = useState<boolean>(false);
  const [tunnelStatus, setTunnelStatus] = useState<string>('disconnected');
  const [showDisclaimer, setShowDisclaimer] = useState<boolean>(true);
  const [acceptedDisclaimer, setAcceptedDisclaimer] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(false);
  const [revealedPassIds, setRevealedPassIds] = useState<string[]>([]);
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
  const [editingPassId, setEditingPassId] = useState<string | null>(null);
  const [newPassword, setNewPassword] = useState<string>('');

  const notify = (msg: string, type: 'info' | 'error' | 'success' = 'info') => {
    if (addToast) {
      addToast(msg, type);
    } else {
      console.log(`[Notification ${type}]:`, msg);
    }
  };

  const fetchShares = async () => {
    try {
      const items = await GetWebShares();
      setShares(items || []);
    } catch (e) {
      console.error(e);
    }
  };

  const initServerInfo = async () => {
    try {
      const ip = await GetLocalIPAddress();
      const p = await GetWebServerPort();
      const running = await IsTunnelRunning();
      const pub = await GetTunnelPublicUrl();

      setLocalIp(ip);
      setPort(p);
      setIsTunneling(running);
      setPublicUrl(pub);
      if (running && pub) {
        setTunnelStatus('connected');
      }
    } catch (e) {
      console.error(e);
    }
  };

  useEffect(() => {
    fetchShares();
    initServerInfo();

    const unsubscribe = EventsOn('tunnel:status', (status: string) => {
      setTunnelStatus(status);
      if (status === 'connected') {
        GetTunnelPublicUrl().then(url => setPublicUrl(url));
        setIsTunneling(true);
      } else if (status === 'disconnected' || status === 'failed') {
        setIsTunneling(false);
        setPublicUrl('');
      }
    });

    return () => {
      if (unsubscribe) unsubscribe();
    };
  }, []);

  const handleDelete = (id: string) => {
    setDeleteConfirmId(id);
  };

  const confirmDelete = async () => {
    if (!deleteConfirmId) return;
    const targetId = deleteConfirmId;
    setDeleteConfirmId(null);
    try {
      const ok = await DeleteWebShare(targetId);
      if (ok) {
        notify(lang === 'id' ? 'Link berbagi berhasil dihapus' : 'Share link deleted successfully', 'success');
        fetchShares();
      }
    } catch (e) {
      notify(String(e), 'error');
    }
  };

  const toggleRevealPass = (id: string) => {
    setRevealedPassIds(prev =>
      prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id]
    );
  };

  const handleSavePassword = async (shareId: string) => {
    try {
      await UpdateWebSharePassword(shareId, newPassword);
      notify(lang === 'id' ? 'Sandi proteksi diperbarui' : 'Password updated successfully', 'success');
      setEditingPassId(null);
      setNewPassword('');
      fetchShares();
    } catch (e) {
      notify(String(e), 'error');
    }
  };

  const handleToggleTunnel = async () => {
    if (!acceptedDisclaimer && !isTunneling) {
      notify(lang === 'id' ? 'Anda harus menyetujui pernyataan disclaimer terlebih dahulu' : 'You must accept the disclaimer first', 'error');
      return;
    }

    setLoading(true);
    try {
      if (isTunneling) {
        await TogglePublicTunnel(false);
        setIsTunneling(false);
        setPublicUrl('');
        setTunnelStatus('disconnected');
        notify(lang === 'id' ? 'Tunnel publik dimatikan' : 'Public tunnel disabled', 'info');
      } else {
        setTunnelStatus('connecting');
        const url = await TogglePublicTunnel(true);
        if (url) {
          setPublicUrl(url);
          setIsTunneling(true);
          setTunnelStatus('connected');
          notify(lang === 'id' ? 'Tunnel publik berhasil diaktifkan!' : 'Public tunnel enabled successfully!', 'success');
        }
      }
    } catch (e) {
      notify(String(e), 'error');
      setTunnelStatus('failed');
    } finally {
      setLoading(false);
    }
  };

  const copyToClipboard = (text: string, typeName: string) => {
    navigator.clipboard.writeText(text);
    notify(lang === 'id' ? `${typeName} disalin ke clipboard` : `${typeName} copied to clipboard`, 'success');
  };

  const getShareLink = (item: WebShareItem, isPub = false) => {
    const host = isPub ? publicUrl : `http://${localIp}:${port}`;
    return `${host}/share/${item.id}`;
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24, color: 'var(--md-sys-color-on-surface, #f8fafc)', maxWidth: 1000, margin: '0 auto', width: '100%', padding: '10px 0' }}>
      
      {/* Top Banner Status */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: 20 }}>
        {/* Local Address card */}
        <div style={{
          padding: 20, borderRadius: 16, background: 'var(--md-sys-color-surface-container-high, rgba(30,41,59,0.7))',
          border: '1.5px solid var(--md-sys-color-outline-variant, rgba(255,255,255,0.1))', display: 'flex', alignItems: 'center', gap: 16
        }}>
          <div style={{
            width: 48, height: 48, borderRadius: '50%', background: 'rgba(52,168,83,0.15)',
            display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#34a853'
          }}>
            <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M5 12.55a11 11 0 0 1 14.08 0"/><path d="M1.42 9a16 16 0 0 1 21.16 0"/><path d="M8.53 16.11a6 6 0 0 1 6.95 0"/><line x1="12" y1="20" x2="12.01" y2="20" strokeWidth="3"/></svg>
          </div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <span style={{ fontSize: 12, fontWeight: 500, color: 'var(--md-sys-color-on-surface-variant, #94a3b8)' }}>
              {lang === 'id' ? 'Alamat Jaringan Lokal' : 'Local Network Address'}
            </span>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 4 }}>
              <span style={{ fontSize: 15, fontWeight: 700, fontFamily: 'monospace', textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap' }}>
                http://{localIp}:{port}
              </span>
              <button
                onClick={() => copyToClipboard(`http://${localIp}:${port}`, 'IP Lokal')}
                style={{ background: 'transparent', border: 'none', cursor: 'pointer', padding: 4, display: 'flex', color: 'var(--md-sys-color-primary, #38bdf8)' }}
                title="Copy Link"
              >
                <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
              </button>
            </div>
          </div>
        </div>

        {/* Public Address Tunnel Card */}
        <div style={{
          padding: 20, borderRadius: 16, background: 'var(--md-sys-color-surface-container-high, rgba(30,41,59,0.7))',
          border: `1.5px solid ${isTunneling ? '#38bdf8' : 'var(--md-sys-color-outline-variant, rgba(255,255,255,0.1))'}`,
          display: 'flex', alignItems: 'center', gap: 16
        }}>
          <div style={{
            width: 48, height: 48, borderRadius: '50%',
            background: isTunneling ? 'rgba(56,189,248,0.15)' : 'rgba(148,163,184,0.15)',
            display: 'flex', alignItems: 'center', justifyContent: 'center', color: isTunneling ? '#38bdf8' : '#94a3b8'
          }}>
            <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
          </div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <span style={{ fontSize: 12, fontWeight: 500, color: 'var(--md-sys-color-on-surface-variant, #94a3b8)' }}>
              {lang === 'id' ? 'Alamat Publik (Cloudflare)' : 'Public Address (Cloudflare)'}
            </span>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 4 }}>
              {isTunneling ? (
                <>
                  <span style={{ fontSize: 14, fontWeight: 700, textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap', color: 'var(--md-sys-color-primary, #38bdf8)' }}>
                    {publicUrl || (lang === 'id' ? 'Menginisialisasi...' : 'Initializing...')}
                  </span>
                  {publicUrl && (
                    <button
                      onClick={() => copyToClipboard(publicUrl, 'URL Publik')}
                      style={{ background: 'transparent', border: 'none', cursor: 'pointer', padding: 4, display: 'flex', color: 'var(--md-sys-color-primary, #38bdf8)' }}
                      title="Copy Link"
                    >
                      <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                    </button>
                  )}
                </>
              ) : (
                <span style={{ fontSize: 13, color: 'var(--md-sys-color-on-surface-variant, #94a3b8)' }}>
                  {lang === 'id' ? 'Tidak aktif' : 'Inactive'}
                </span>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Safety Disclaimer Panel */}
      {!isTunneling && showDisclaimer && (
        <div style={{
          padding: 24, borderRadius: 20, background: 'rgba(244,63,94,0.08)',
          border: '1.5px solid rgba(244,63,94,0.3)', display: 'flex', flexDirection: 'column', gap: 16
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, color: '#f43f5e' }}>
            <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16" strokeWidth="3"/></svg>
            <h3 style={{ fontSize: 16, fontWeight: 700, margin: 0 }}>
              {lang === 'id' ? 'Pernyataan Keamanan Berbagi Publik' : 'Public Sharing Security Disclaimer'}
            </h3>
          </div>
          <div style={{ fontSize: 13, lineHeight: 1.6, color: 'var(--md-sys-color-on-surface-variant, #94a3b8)' }}>
            {lang === 'id' ? (
              <ul style={{ margin: 0, paddingLeft: 20 }}>
                <li>Mengaktifkan tunnel publik (Cloudflare) membolehkan siapa saja yang memiliki link untuk mengakses/mengunduh berkas yang Anda bagikan secara langsung melalui internet.</li>
                <li>Komputer Anda <b>harus tetap menyala</b> dan aplikasi Awd DriveRouter Desktop harus aktif agar link web sharing dapat terus diakses.</li>
                <li>Proses unduhan pengunjung akan menggunakan <b>bandwidth upload internet rumah Anda</b>.</li>
                <li>Sangat disarankan menyetel <b>Password Proteksi</b> untuk dokumen yang sensitif.</li>
              </ul>
            ) : (
              <ul style={{ margin: 0, paddingLeft: 20 }}>
                <li>Enabling public tunnel (Cloudflare) allows anyone with the link to access/download shared items directly from the internet.</li>
                <li>Your computer <b>must remain turned on</b> and Awd DriveRouter Desktop must be running for the sharing link to be accessible.</li>
                <li>Download speed for visitors depends on your <b>home internet upload bandwidth</b>.</li>
                <li>It is highly recommended to set a <b>Password Protection</b> for sensitive documents.</li>
              </ul>
            )}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginTop: 8 }}>
            <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, fontWeight: 600, cursor: 'pointer' }}>
              <input
                type="checkbox"
                checked={acceptedDisclaimer}
                onChange={e => setAcceptedDisclaimer(e.target.checked)}
                style={{ width: 18, height: 18, cursor: 'pointer' }}
              />
              {lang === 'id' ? 'Saya memahami dan menyetujui konsekuensi keamanan di atas' : 'I understand and agree to the security consequences above'}
            </label>
          </div>
        </div>
      )}

      {/* Control Action Panel */}
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '16px 24px', borderRadius: 16, background: 'var(--md-sys-color-surface-container-high, rgba(30,41,59,0.7))',
        border: '1px solid var(--md-sys-color-outline-variant, rgba(255,255,255,0.1))'
      }}>
        <div>
          <h3 style={{ fontSize: 15, fontWeight: 700, margin: 0 }}>
            {lang === 'id' ? 'Tunnel Publik Cloudflare' : 'Cloudflare Public Tunnel'}
          </h3>
          <p style={{ fontSize: 12, color: 'var(--md-sys-color-on-surface-variant, #94a3b8)', marginTop: 4 }}>
            {tunnelStatus === 'downloading' && (lang === 'id' ? 'Mengunduh cloudflared runner... Mohon tunggu' : 'Downloading cloudflared runner... Please wait')}
            {tunnelStatus === 'connecting' && (lang === 'id' ? 'Menghubungkan tunnel ke Cloudflare...' : 'Connecting tunnel to Cloudflare...')}
            {tunnelStatus === 'connected' && (lang === 'id' ? 'Tunnel aktif berjalan' : 'Tunnel is running successfully')}
            {tunnelStatus === 'disconnected' && (lang === 'id' ? 'Tunnel publik tidak aktif' : 'Public tunnel is disabled')}
            {tunnelStatus === 'failed' && (lang === 'id' ? 'Gagal menghubungkan tunnel. Silakan coba lagi.' : 'Tunnel connection failed. Please retry.')}
          </p>
        </div>
        <button
          onClick={handleToggleTunnel}
          disabled={loading || tunnelStatus === 'downloading' || tunnelStatus === 'connecting'}
          style={{
            padding: '10px 24px', borderRadius: 100, border: 'none',
            background: isTunneling ? '#f43f5e' : 'var(--md-sys-color-primary, #38bdf8)',
            color: isTunneling ? '#ffffff' : '#0f172a', fontSize: 13, fontWeight: 700, cursor: 'pointer',
            display: 'flex', alignItems: 'center', gap: 8, transition: 'filter .15s'
          }}
        >
          {isTunneling ? (lang === 'id' ? 'Matikan Publik' : 'Disable Public') : (lang === 'id' ? 'Aktifkan Publik' : 'Enable Public')}
        </button>
      </div>

      {/* Shared Items Table */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <h3 style={{ fontSize: 18, fontWeight: 600 }}>
          {lang === 'id' ? 'Manajemen Web Sharing' : 'Web Share Management'}
        </h3>
        
        {shares.length === 0 ? (
          <div style={{
            padding: 48, borderRadius: 16, border: '1.5px dashed var(--md-sys-color-outline-variant, rgba(255,255,255,0.15))',
            display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12, textAlign: 'center'
          }}>
            <svg viewBox="0 0 24 24" width="44" height="44" fill="none" stroke="currentColor" strokeWidth="1.5" style={{ color: 'var(--md-sys-color-on-surface-variant, #94a3b8)' }}><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/></svg>
            <span style={{ fontSize: 14, fontWeight: 600, color: 'var(--md-sys-color-on-surface-variant, #94a3b8)' }}>
              {lang === 'id' ? 'Belum ada file atau folder yang dibagikan' : 'No files or folders are currently shared'}
            </span>
            <span style={{ fontSize: 12, color: 'var(--md-sys-color-on-surface-variant, #64748b)' }}>
              {lang === 'id' ? 'Klik kanan pada file di Explorer lalu pilih "Berbagi Web (Lokal/Public)"' : 'Right-click any file in Explorer and select "Create Web Share"'}
            </span>
          </div>
        ) : (
          <div style={{
            background: 'var(--md-sys-color-surface-container-high, rgba(30,41,59,0.7))',
            border: '1px solid var(--md-sys-color-outline-variant, rgba(255,255,255,0.1))',
            borderRadius: 16, overflow: 'hidden'
          }}>
            <div style={{ overflowX: 'auto' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid var(--md-sys-color-outline-variant, rgba(255,255,255,0.1))', textAlign: 'left', color: 'var(--md-sys-color-on-surface-variant, #94a3b8)' }}>
                    <th style={{ padding: '14px 16px', fontWeight: 600 }}>{lang === 'id' ? 'Nama Berkas' : 'Name'}</th>
                    <th style={{ padding: '14px 16px', fontWeight: 600 }}>{lang === 'id' ? 'Tipe' : 'Type'}</th>
                    <th style={{ padding: '14px 16px', fontWeight: 600 }}>{lang === 'id' ? 'Ukuran' : 'Size'}</th>
                    <th style={{ padding: '14px 16px', fontWeight: 600 }}>{lang === 'id' ? 'Akses' : 'Visits'}</th>
                    <th style={{ padding: '14px 16px', fontWeight: 600 }}>{lang === 'id' ? 'Password' : 'Password'}</th>
                    <th style={{ padding: '14px 16px', fontWeight: 600, textAlign: 'right' }}>{lang === 'id' ? 'Aksi' : 'Actions'}</th>
                  </tr>
                </thead>
                <tbody>
                  {shares.map(item => (
                    <tr key={item.id} style={{ borderBottom: '1px solid var(--md-sys-color-outline-variant, rgba(255,255,255,0.06))' }}>
                      <td style={{ padding: '14px 16px', fontWeight: 600 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                          {item.type === 'folder' ? (
                            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="#38bdf8" strokeWidth="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
                          ) : (
                            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="#94a3b8" strokeWidth="2"><path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><polyline points="13 2 13 9 20 9"/></svg>
                          )}
                          <span style={{ textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap', maxWidth: 220 }}>
                            {item.name}
                          </span>
                        </div>
                      </td>
                      <td style={{ padding: '14px 16px', textTransform: 'capitalize', color: 'var(--md-sys-color-on-surface-variant, #94a3b8)' }}>
                        {item.type}
                      </td>
                      <td style={{ padding: '14px 16px', color: 'var(--md-sys-color-on-surface-variant, #94a3b8)' }}>
                        {item.type === 'folder' ? '-' : formatBytes(item.size)}
                      </td>
                      <td style={{ padding: '14px 16px', color: 'var(--md-sys-color-on-surface-variant, #94a3b8)' }}>
                        {item.accessCount || 0}
                      </td>
                      <td style={{ padding: '14px 16px' }}>
                        {editingPassId === item.id ? (
                          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                            <input
                              type="text"
                              value={newPassword}
                              onChange={e => setNewPassword(e.target.value)}
                              placeholder="New pass"
                              style={{
                                width: 100, padding: '4px 8px', borderRadius: 6,
                                border: '1px solid var(--md-sys-color-outline-variant, rgba(255,255,255,0.2))',
                                background: 'rgba(0,0,0,0.3)', color: '#fff', fontSize: 12
                              }}
                            />
                            <button
                              onClick={() => handleSavePassword(item.id)}
                              style={{ padding: '4px 8px', borderRadius: 6, border: 'none', background: '#38bdf8', color: '#0f172a', fontWeight: 700, fontSize: 11, cursor: 'pointer' }}
                            >
                              Save
                            </button>
                            <button
                              onClick={() => setEditingPassId(null)}
                              style={{ padding: '4px 8px', borderRadius: 6, border: 'none', background: 'transparent', color: '#94a3b8', fontSize: 11, cursor: 'pointer' }}
                            >
                              X
                            </button>
                          </div>
                        ) : (
                          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                            {item.password ? (
                              <span style={{
                                padding: '2px 8px', borderRadius: 100, background: 'rgba(56,189,248,0.15)',
                                color: '#38bdf8', fontSize: 12, fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: 4
                              }}>
                                <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                                {revealedPassIds.includes(item.id) ? item.password : '••••••'}
                              </span>
                            ) : (
                              <span style={{ fontSize: 12, color: '#64748b' }}>None</span>
                            )}
                            {item.password && (
                              <button
                                onClick={() => toggleRevealPass(item.id)}
                                style={{ background: 'transparent', border: 'none', cursor: 'pointer', padding: 2, display: 'inline-flex', color: '#94a3b8' }}
                                title="Toggle View"
                              >
                                {revealedPassIds.includes(item.id) ? (
                                  <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24M1 1l22 22"/></svg>
                                ) : (
                                  <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                                )}
                              </button>
                            )}
                            <button
                              onClick={() => { setEditingPassId(item.id); setNewPassword(item.password || ''); }}
                              style={{ background: 'transparent', border: 'none', cursor: 'pointer', padding: 2, display: 'inline-flex', color: '#38bdf8' }}
                              title="Edit Password"
                            >
                              <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                            </button>
                          </div>
                        )}
                      </td>
                      <td style={{ padding: '14px 16px', textAlign: 'right' }}>
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: 8 }}>
                          <button
                            onClick={() => copyToClipboard(getShareLink(item, false), 'Link Lokal')}
                            style={{
                              padding: '6px 12px', borderRadius: 8, border: '1px solid var(--md-sys-color-outline-variant, rgba(255,255,255,0.15))',
                              background: 'transparent', color: '#38bdf8', fontSize: 12, fontWeight: 600, cursor: 'pointer'
                            }}
                          >
                            Copy Local
                          </button>
                          {isTunneling && publicUrl && (
                            <button
                              onClick={() => copyToClipboard(getShareLink(item, true), 'Link Publik')}
                              style={{
                                padding: '6px 12px', borderRadius: 8, border: 'none',
                                background: '#38bdf8', color: '#0f172a', fontSize: 12, fontWeight: 700, cursor: 'pointer'
                              }}
                            >
                              Copy Public
                            </button>
                          )}
                          <button
                            onClick={() => handleDelete(item.id)}
                            style={{
                              padding: '6px 10px', borderRadius: 8, border: 'none',
                              background: 'rgba(244,63,94,0.15)', color: '#f43f5e', fontSize: 12, fontWeight: 600, cursor: 'pointer'
                            }}
                            title="Delete Share"
                          >
                            Delete
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>

      {/* Delete Confirmation Modal */}
      {deleteConfirmId && (
        <div style={{
          position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
          background: 'rgba(0,0,0,0.6)', backdropFilter: 'blur(4px)',
          display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 9999
        }}>
          <div style={{
            background: 'var(--md-sys-color-surface-container-high, #1e293b)',
            border: '1px solid var(--md-sys-color-outline-variant, rgba(255,255,255,0.1))',
            borderRadius: 20, padding: 28, width: 360, textAlign: 'center', display: 'flex', flexDirection: 'column', gap: 16
          }}>
            <h3 style={{ margin: 0, fontSize: 17, fontWeight: 700 }}>
              {lang === 'id' ? 'Hapus Link Berbagi?' : 'Delete Share Link?'}
            </h3>
            <p style={{ margin: 0, fontSize: 13, color: 'var(--md-sys-color-on-surface-variant, #94a3b8)' }}>
              {lang === 'id' ? 'Tautan ini tidak akan dapat diakses lagi melalui web.' : 'This share link will no longer be accessible via web.'}
            </p>
            <div style={{ display: 'flex', gap: 12, justifyContent: 'center', marginTop: 8 }}>
              <button
                onClick={() => setDeleteConfirmId(null)}
                style={{
                  flex: 1, padding: '10px 16px', borderRadius: 100, border: '1px solid var(--md-sys-color-outline-variant, rgba(255,255,255,0.2))',
                  background: 'transparent', color: '#fff', fontSize: 13, cursor: 'pointer'
                }}
              >
                {lang === 'id' ? 'Batal' : 'Cancel'}
              </button>
              <button
                onClick={confirmDelete}
                style={{
                  flex: 1, padding: '10px 16px', borderRadius: 100, border: 'none',
                  background: '#f43f5e', color: '#fff', fontSize: 13, fontWeight: 700, cursor: 'pointer'
                }}
              >
                {lang === 'id' ? 'Hapus' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}
