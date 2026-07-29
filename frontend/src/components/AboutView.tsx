import React, { useState, useEffect } from 'react';
import changelogData from '../locales/changelog.json';
import logoImg from '../assets/images/logo.png';
// @ts-ignore
import { CheckForUpdates, OpenReleaseURL, GetAppVersion } from '../../wailsjs/go/main/App';

interface AboutViewProps {
  lang: 'en' | 'id';
}

export default function AboutView({ lang }: AboutViewProps) {
  const [checking, setChecking] = useState(false);
  const [updateResult, setUpdateResult] = useState<any>(null);
  const [errorMsg, setErrorMsg] = useState('');
  const [showLicense, setShowLicense] = useState(false);
  const [appVersion, setAppVersion] = useState('1.1.0');

  useEffect(() => {
    GetAppVersion()
      .then((ver: string) => {
        if (ver) setAppVersion(ver);
      })
      .catch((err: any) => console.error(err));
  }, []);

  const handleManualCheck = async () => {
    setChecking(true);
    setUpdateResult(null);
    setErrorMsg('');
    try {
      const info = await CheckForUpdates();
      setUpdateResult(info);
    } catch (err: any) {
      console.error(err);
      setErrorMsg(lang === 'id' ? 'Gagal melakukan pengecekan pembaruan.' : 'Failed to check for updates.');
    } finally {
      setChecking(false);
    }
  };

  const mitLicenseText = lang === 'id' 
    ? `Hak Cipta © 2026 I Putu Agus Wahyu Dupayana\n\nIzin dengan ini diberikan, secara gratis, kepada siapa pun yang memperoleh salinan perangkat lunak ini dan file dokumentasi terkait ("Perangkat Lunak"), untuk menggunakan Perangkat Lunak tanpa batasan, termasuk tanpa batasan hak untuk menggunakan, menyalin, memodifikasi, menggabungkan, menerbitkan, mendistribusikan, mensublisensikan, dan/atau menjual salinan Perangkat Lunak, dan mengizinkan orang yang menerima Perangkat Lunak untuk melakukannya, dengan tunduk pada ketentuan berikut:\n\nPemberitahuan hak cipta di atas dan pemberitahuan izin ini harus dicantumkan dalam semua salinan atau bagian substansial dari Perangkat Lunak.\n\nPERANGKAT LUNAK INI DISEDIAKAN "APA ADANYA", TANPA JAMINAN APA PUN, TERSURAT MAUPUN TERSIRAT, TERMASUK NAMUN TIDAK TERBATAS PADA JAMINAN KELAYAKAN JUAL, KESESUAIAN UNTUK TUJUAN TERTENTU DAN KETIADAAN PELANGGARAN. DALAM HAL APA PUN, PENULIS ATAU PEMEGANG HAK CIPTA TIDAK BERTANGGUNG JAWAB ATAS KLAIM, KERUSAKAN ATAU KEWAJIBAN LAINNYA, BAIK DALAM TINDAKAN KONTRAK, PERBUATAN MELAWAN HUKUM ATAU LAINNYA, YANG TIMBUL DARI, DARI ATAU SEHUBUNGAN DENGAN PERANGKAT LUNAK ATAU PENGGUNAAN ATAU TRANSAKSI LAIN DALAM PERANGKAT LUNAK.`
    : `Copyright © 2026 I Putu Agus Wahyu Dupayana\n\nPermission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:\n\nThe above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.\n\nTHE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.`;  return (
    <div style={{
      maxWidth: 850, margin: '0 auto', fontFamily: 'var(--font-family)',
      display: 'flex', flexDirection: 'column', gap: 24, paddingBottom: 40
    }}>
      {/* Header Info */}
      <div style={{
        background: 'var(--md-sys-color-surface-container-high)',
        borderRadius: 24, padding: 24, display: 'flex', alignItems: 'center', gap: 20,
        border: '1px solid var(--md-sys-color-outline-variant)', flexWrap: 'wrap'
      }}>
        <img src={logoImg} alt="Awd DriveRouter" style={{ width: 72, height: 72, borderRadius: 16, objectFit: 'contain' }} />
        <div style={{ flex: 1, minWidth: 200 }}>
          <h3 style={{ margin: '0 0 6px 0', fontSize: 22, color: 'var(--md-sys-color-on-surface)' }}>Awd DriveRouter</h3>
          <p style={{ margin: '0 0 8px 0', fontSize: 14, color: 'var(--md-sys-color-on-surface-variant)' }}>
            {lang === 'id' 
              ? 'Aplikasi Desktop penggabung & pengatur alokasi penyimpanan multi-cloud secara cerdas.' 
              : 'Desktop application to manage and route multi-cloud storage allocation seamlessly.'}
          </p>
          <div style={{ display: 'flex', gap: 12, alignItems: 'center', fontSize: 13, color: 'var(--md-sys-color-on-surface-variant)' }}>
            <span>{lang === 'id' ? `Versi saat ini: ${appVersion}` : `Current version: ${appVersion}`}</span>
            <span style={{ height: 4, width: 4, borderRadius: '50%', background: 'var(--md-sys-color-outline)' }}></span>
            <span>© 2026 Awd DriveRouter</span>
          </div>
        </div>
        <div>
          <button
            onClick={handleManualCheck}
            disabled={checking}
            style={{
              display: 'flex', alignItems: 'center', gap: 8,
              background: 'var(--md-sys-color-primary)', color: 'var(--md-sys-color-on-primary)',
              border: 'none', padding: '10px 20px', borderRadius: 100,
              fontSize: 14, fontWeight: 600, cursor: checking ? 'not-allowed' : 'pointer',
              opacity: checking ? 0.8 : 1, transition: 'all 0.2s ease',
              boxShadow: 'var(--shadow-1)'
            }}
            onMouseEnter={e => { if(!checking) e.currentTarget.style.filter = 'brightness(0.9)'; }}
            onMouseLeave={e => { if(!checking) e.currentTarget.style.filter = 'none'; }}
          >
            <svg
              width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"
              style={{ animation: checking ? 'spin 1s linear infinite' : 'none' }}
            >
              <path d="M23 4v6h-6" />
              <path d="M1 20v-6h6" />
              <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
            </svg>
            {lang === 'id' ? 'Periksa Pembaruan' : 'Check for Updates'}
          </button>
        </div>
      </div>

      <style>{`
        @keyframes spin { to { transform: rotate(360deg); } }
        @keyframes fadeIn { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: translateY(0); } }
      `}</style>

      {/* Manual Check Result */}
      {errorMsg && (
        <div style={{
          display: 'flex', alignItems: 'center', gap: 10, padding: '12px 16px',
          background: 'var(--md-sys-color-error-container)', color: 'var(--md-sys-color-on-error-container)',
          borderRadius: 12, fontSize: 14, border: '1px solid var(--md-sys-color-error)'
        }}>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
          {errorMsg}
        </div>
      )}

      {updateResult && (
        <div style={{
          padding: '16px 20px', borderRadius: 16,
          background: updateResult.has_update ? 'var(--md-sys-color-primary-container)' : 'var(--md-sys-color-success-container)',
          border: updateResult.has_update ? '1px solid var(--md-sys-color-primary)' : '1px solid var(--md-sys-color-success)',
          color: updateResult.has_update ? 'var(--md-sys-color-on-primary-container)' : 'var(--md-sys-color-on-success-container)',
          display: 'flex', flexDirection: 'column', gap: 8, fontSize: 14,
          animation: 'fadeIn 0.2s ease-out'
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            {updateResult.has_update ? (
              <>
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
                <strong>
                  {lang === 'id' 
                    ? `Pembaruan tersedia! Versi terbaru: ${updateResult.latest_version}` 
                    : `Update available! Latest version: ${updateResult.latest_version}`}
                </strong>
              </>
            ) : (
              <>
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
                <strong>
                  {lang === 'id' ? 'Aplikasi Anda sudah menggunakan versi terbaru.' : 'Your application is up to date.'}
                </strong>
              </>
            )}
          </div>
          {updateResult.has_update && (
            <div style={{ display: 'flex', justifyContent: 'flex-start', marginTop: 4 }}>
              <button
                onClick={() => OpenReleaseURL(updateResult.update_url)}
                style={{
                  background: 'var(--md-sys-color-primary)', color: 'var(--md-sys-color-on-primary)', border: 'none',
                  padding: '6px 16px', borderRadius: 100, fontSize: 13, fontWeight: 500, cursor: 'pointer'
                }}
              >
                {lang === 'id' ? 'Unduh Pembaruan' : 'Download Update'}
              </button>
            </div>
          )}
        </div>
      )}

      {/* Cards Links & Legal */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))', gap: 16 }}>
        {/* Official Links */}
        <div style={{
          background: 'var(--md-sys-color-surface-container)',
          borderRadius: 20, padding: 20, border: '1px solid var(--md-sys-color-outline-variant)'
        }}>
          <h4 style={{ margin: '0 0 14px 0', fontSize: 16, fontWeight: 600, color: 'var(--md-sys-color-on-surface)' }}>
            {lang === 'id' ? 'Tautan Resmi' : 'Official Links'}
          </h4>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            <div 
              onClick={() => OpenReleaseURL('https://github.com/putuwahyu29/awd-driverouter-desktop')}
              style={{ display: 'flex', alignItems: 'center', gap: 12, cursor: 'pointer' }}
            >
              <div style={{
                background: 'var(--md-sys-color-surface-container-high)', borderRadius: '50%',
                width: 38, height: 38, display: 'flex', alignItems: 'center', justifyContent: 'center',
                color: 'var(--md-sys-color-primary)'
              }}>
                <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/></svg>
              </div>
              <div>
                <div style={{ fontSize: 14, fontWeight: 500, color: 'var(--md-sys-color-on-surface)' }}>
                  {lang === 'id' ? 'GitHub Repository' : 'GitHub Repository'}
                </div>
                <div style={{ fontSize: 12, color: 'var(--md-sys-color-on-surface-variant)' }}>putuwahyu29/awd-driverouter-desktop</div>
              </div>
            </div>

            <div 
              onClick={() => OpenReleaseURL('https://github.com/putuwahyu29/awd-driverouter-desktop/issues')}
              style={{ display: 'flex', alignItems: 'center', gap: 12, cursor: 'pointer' }}
            >
              <div style={{
                background: 'var(--md-sys-color-surface-container-high)', borderRadius: '50%',
                width: 38, height: 38, display: 'flex', alignItems: 'center', justifyContent: 'center',
                color: 'var(--md-sys-color-primary)'
              }}>
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="17"/></svg>
              </div>
              <div>
                <div style={{ fontSize: 14, fontWeight: 500, color: 'var(--md-sys-color-on-surface)' }}>
                  {lang === 'id' ? 'Laporkan Masalah' : 'Report Issues'}
                </div>
                <div style={{ fontSize: 12, color: 'var(--md-sys-color-on-surface-variant)' }}>github.com/putuwahyu29/awd-driverouter-desktop/issues</div>
              </div>
            </div>
          </div>
        </div>

        {/* Legal & Creator */}
        <div style={{
          background: 'var(--md-sys-color-surface-container)',
          borderRadius: 20, padding: 20, border: '1px solid var(--md-sys-color-outline-variant)'
        }}>
          <h4 style={{ margin: '0 0 14px 0', fontSize: 16, fontWeight: 600, color: 'var(--md-sys-color-on-surface)' }}>
            {lang === 'id' ? 'Legal & Pembuat' : 'Legal & Creator'}
          </h4>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            <div 
              onClick={() => OpenReleaseURL('https://github.com/putuwahyu29')}
              style={{ display: 'flex', alignItems: 'center', gap: 12, cursor: 'pointer' }}
            >
              <div style={{
                background: 'var(--md-sys-color-surface-container-high)', borderRadius: '50%',
                width: 38, height: 38, display: 'flex', alignItems: 'center', justifyContent: 'center',
                color: 'var(--md-sys-color-primary)'
              }}>
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
              </div>
              <div>
                <div style={{ fontSize: 14, fontWeight: 500, color: 'var(--md-sys-color-on-surface)' }}>
                  {lang === 'id' ? 'Pembuat / Developer' : 'Creator / Developer'}
                </div>
                <div style={{ fontSize: 12, color: 'var(--md-sys-color-on-surface-variant)' }}>I Putu Agus Wahyu Dupayana</div>
              </div>
            </div>

            <div 
              onClick={() => setShowLicense(!showLicense)}
              style={{ display: 'flex', alignItems: 'center', gap: 12, cursor: 'pointer' }}
            >
              <div style={{
                background: 'var(--md-sys-color-surface-container-high)', borderRadius: '50%',
                width: 38, height: 38, display: 'flex', alignItems: 'center', justifyContent: 'center',
                color: 'var(--md-sys-color-primary)'
              }}>
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
              </div>
              <div>
                <div style={{ fontSize: 14, fontWeight: 500, color: 'var(--md-sys-color-on-surface)' }}>
                  {lang === 'id' ? 'Lisensi Perangkat Lunak' : 'Software License'}
                </div>
                <div style={{ fontSize: 12, color: 'var(--md-sys-color-on-surface-variant)' }}>MIT License</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* MIT License Expander */}
      {showLicense && (
        <div style={{
          background: 'var(--md-sys-color-surface-container)',
          borderRadius: 20, padding: 20, border: '1px solid var(--md-sys-color-outline-variant)',
          fontSize: 12, color: 'var(--md-sys-color-on-surface-variant)', lineHeight: '1.6',
          whiteSpace: 'pre-wrap', maxHeight: 200, overflowY: 'auto',
          animation: 'fadeIn 0.2s ease-out'
        }}>
          <strong style={{ color: 'var(--md-sys-color-on-surface)' }}>MIT License</strong>
          <br /><br />
          {mitLicenseText}
        </div>
      )}

      {/* Changelog Timeline */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <h4 style={{ margin: '8px 0 0 0', fontSize: 18, color: 'var(--md-sys-color-on-surface)' }}>
          {lang === 'id' ? 'Catatan Rilis (Changelog)' : 'Release Notes (Changelog)'}
        </h4>

        {changelogData.map((item: any, index: number) => (
          <div key={item.version} style={{ display: 'flex', gap: 16, position: 'relative' }}>
            {/* Dot & Line */}
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
              <div style={{
                width: 12, height: 12, borderRadius: '50%',
                background: index === 0 ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-outline)',
                marginTop: 6, zIndex: 2
              }} />
              {index < changelogData.length - 1 && (
                <div style={{
                  width: 2, flex: 1, background: 'var(--md-sys-color-outline-variant)',
                  marginTop: 4, marginBottom: -6, zIndex: 1
                }} />
              )}
            </div>

            {/* Content Card */}
            <div style={{
              flex: 1, background: 'var(--md-sys-color-surface-container)',
              borderRadius: 16, padding: 18, border: '1px solid var(--md-sys-color-outline-variant)',
              marginBottom: 10
            }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8, flexWrap: 'wrap', gap: 8 }}>
                <span style={{ fontSize: 16, fontWeight: 600, color: 'var(--md-sys-color-on-surface)' }}>
                  {item.title[lang] || item.title.en}
                </span>
                <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                  <span style={{
                    fontSize: 12, fontWeight: 600,
                    background: index === 0 ? 'var(--md-sys-color-primary-container)' : 'transparent',
                    color: index === 0 ? 'var(--md-sys-color-on-primary-container)' : 'var(--md-sys-color-on-surface-variant)',
                    padding: '2px 8px', borderRadius: 6, border: '1px solid var(--md-sys-color-outline-variant)'
                  }}>
                    v{item.version}
                  </span>
                  <span style={{ fontSize: 12, color: 'var(--md-sys-color-on-surface-variant)' }}>{item.date}</span>
                </div>
              </div>

              <ul style={{ margin: 0, paddingLeft: 20, fontSize: 13, lineHeight: '1.6', color: 'var(--md-sys-color-on-surface-variant)' }}>
                {(item.changes[lang] || item.changes.en).map((change: string, i: number) => (
                  <li key={i} style={{ marginBottom: 4 }}>{change}</li>
                ))}
              </ul>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
