import React from 'react';

// Inline SVG Icons for Material Design 3 and Cloud Logos
export const IconHome = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/></svg>;
export const IconFolder = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/></svg>;
export const IconFile = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M14 2H6c-1.1 0-1.99.9-1.99 2L4 20c0 1.1.89 2 1.99 2H18c1.1 0 2-.9 2-2V8l-6-6zm2 16H8v-2h8v2zm0-4H8v-2h8v2zm-3-5V3.5L18.5 9H13z"/></svg>;
export const IconStar = ({ filled }: { filled?: boolean }) => <svg width="20" height="20" viewBox="0 0 24 24" fill={filled ? "#f4b400" : "currentColor"} stroke={filled ? "#f4b400" : "none"}><path d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z"/></svg>;
export const IconSettings = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.07-.94l2.03-1.58c.18-.14.23-.41.12-.61l-1.92-3.32c-.12-.22-.37-.29-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54c-.04-.24-.24-.41-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.09.63-.09.94s.02.64.07.94l-2.03 1.58c-.18.14-.23.41-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"/></svg>;
export const IconCloud = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96z"/></svg>;
export const IconSearch = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M15.5 14h-.79l-.28-.27C15.41 12.59 16 11.11 16 9.5 16 5.91 13.09 3 9.5 3S3 5.91 3 9.5 5.91 16 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/></svg>;
export const IconPlus = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
    <path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
  </svg>
);
export const IconDots = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M12 8c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm0 2c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm0 6c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z"/></svg>;
export const IconDelete = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/></svg>;
export const IconRename = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z"/></svg>;
export const IconDownload = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM17 13l-5 5-5-5h3V9h4v4h3z"/></svg>;
export const IconInfo = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/></svg>;
export const IconChevronRight = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M10 6L8.59 7.41 13.17 12l-4.58 4.59L10 18l6-6z"/></svg>;
export const IconClose = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg>;
export const IconRefresh = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/></svg>;
export const IconWarning = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/></svg>;

export const IconGoogleDrive = () => (
  <svg viewBox="0 0 87.3 78" width="24" height="24" xmlns="http://www.w3.org/2000/svg">
    {/* Blue left arm */}
    <path d="m6.6 66.85 3.85 6.65c.8 1.4 1.95 2.5 3.3 3.3l13.75-23.8h-27.5c0 1.55.4 3.1 1.2 4.5z" fill="#0066da"/>
    {/* Green left column */}
    <path d="m43.65 25-13.75-23.8c-1.35.8-2.5 1.9-3.3 3.3l-25.4 44a9.06 9.06 0 0 0 -1.2 4.5h27.5z" fill="#00ac47"/>
    {/* Red right arm */}
    <path d="m73.55 76.8c1.35-.8 2.5-1.9 3.3-3.3l1.6-2.75 7.65-13.25c.8-1.4 1.2-2.95 1.2-4.5h-27.502l5.852 11.5z" fill="#ea4335"/>
    {/* Dark green top right */}
    <path d="m43.65 25 13.75-23.8c-1.35-.8-2.9-1.2-4.5-1.2h-18.5c-1.6 0-3.15.45-4.5 1.2z" fill="#00832d"/>
    {/* Blue bottom bar */}
    <path d="m59.8 53h-32.3l-13.75 23.8c1.35.8 2.9 1.2 4.5 1.2h50.8c1.6 0 3.15-.45 4.5-1.2z" fill="#2684fc"/>
    {/* Yellow right column */}
    <path d="m73.4 26.5-12.7-22c-.8-1.4-1.95-2.5-3.3-3.3l-13.75 23.8 16.15 27h27.45c0-1.55-.4-3.1-1.2-4.5z" fill="#ffba00"/>
  </svg>
);

export const IconOneDrive = () => (
  <svg viewBox="0 0 24 24" width="24" height="24" fill="#0078d4">
    <path d="M19.5 10.5c-1.38 0-2.5 1.12-2.5 2.5 0 .14.02.28.05.41C16.14 12.44 14.68 11.5 13 11.5c-2.3 0-4.22 1.74-4.48 4.02-.4-.25-.88-.4-1.39-.4-1.45 0-2.63 1.25-2.63 2.8s1.18 2.8 2.63 2.8h12.37c2.49 0 4.5-2.01 4.5-4.5s-2.01-4.5-4.5-4.5zM6.5 12c.28 0 .54.04.8.12C8.04 10.28 9.85 9 12 9c2.34 0 4.31 1.48 5.09 3.63.48-.41 1.1-.63 1.76-.63 1.55 0 2.8 1.26 2.8 2.8 0 .07 0 .13-.01.2C23.01 12.8 21.19 11 19 11c-.42 0-.82.07-1.2.2-.64-2.5-2.9-4.2-5.46-4.2-2.22 0-4.2 1.28-5.1 3.25-.43-.16-.9-.25-1.39-.25-1.99 0-3.64 1.5-3.83 3.44C1.34 12.59 3.68 12 6.5 12z"/>
  </svg>
);

export const IconDropbox = () => (
  <svg viewBox="0 0 24 24" width="24" height="24" fill="#0061ff">
    <path d="M6 2l6 4-6 4-6-4zm12 0l6 4-6 4-6-4zm-12 8l6 4-6 4-6-4zm12 0l6 4-6 4-6-4zM6 18.5l6 4 6-4v-1.5H6z"/>
  </svg>
);

export const IconBox = () => (
  <svg viewBox="0 0 24 24" width="24" height="24" xmlns="http://www.w3.org/2000/svg">
    <path fill="#0061d5" d="M5 2C3.34 2 2 3.34 2 5v14c0 1.66 1.34 3 3 3h14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3H5zm1.5 5h2v3.5c.45-.27.96-.5 1.5-.5 1.93 0 3.5 1.57 3.5 3.5S11.93 17 10 17c-.54 0-1.05-.23-1.5-.5V17h-2V7zm4 5c-.83 0-1.5.67-1.5 1.5S9.67 15 10.5 15s1.5-.67 1.5-1.5S11.33 12 10.5 12zm5-5c1.93 0 3.5 1.57 3.5 3.5S17.43 14 15.5 14c-.54 0-1.05-.23-1.5-.5V17h-2v-9.5h2v.5c.45-.27.96-.5 1.5-.5zm0 2c-.83 0-1.5.67-1.5 1.5S14.67 12 15.5 12s1.5-.67 1.5-1.5S16.33 9 15.5 9z"/>
  </svg>
);

export const IconYandex = () => (
  <svg viewBox="0 0 24 24" width="24" height="24">
    <path fill="#fc3f35" d="M12.5 2L6 22h3.5l1.5-6h4.5l1.5 6H21L14.5 2h-2zM12 12l1.5-6.5L15 12h-3z"/>
  </svg>
);

export const IconPCloud = () => (
  <svg viewBox="0 0 256 221" width="24" height="24" xmlns="http://www.w3.org/2000/svg">
    <path fill="#3F9BDB" d="M205.9 221H50.1C22.4 221 0 198.6 0 170.9c0-22.6 14.9-41.8 35.5-48.2C34 118 33.2 113 33.2 107.8c0-42.4 34.4-76.8 76.8-76.8 24.1 0 45.6 11.1 59.7 28.5 5.8-2.2 12.1-3.5 18.7-3.5 29.5 0 53.4 23.9 53.4 53.4 0 2.7-.2 5.4-.6 8 8.9 6.8 14.8 17.5 14.8 29.7 0 20.3-16.5 36.8-36.8 36.8H205.9V221z"/>
    <path fill="white" d="M95.3 131.8V168h18.1v-16.5h22.3c17.7 0 28.7-10.4 28.7-26.1 0-14.9-10.3-25.1-26.8-25.1H95.3v31.5zm18.1-16.6h18.4c7.1 0 11.4 3.5 11.4 9.7 0 6.1-4.3 9.7-11.4 9.7h-18.4v-19.4z"/>
  </svg>
);

export const IconTelegram = () => (
  <svg viewBox="0 0 24 24" width="24" height="24" fill="#0088cc">
    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm4.64 6.8c-.15 1.58-.8 5.42-1.13 7.19-.14.75-.42 1-.68 1.03-.58.05-1.02-.38-1.58-.75-.88-.58-1.38-.94-2.23-1.5-.99-.65-.35-1 .22-1.59.15-.15 2.71-2.48 2.76-2.69a.2.2 0 0 0-.05-.18c-.06-.05-.14-.03-.21-.02-.09.02-1.49.95-4.22 2.79-.4.27-.76.41-1.08.4-.36-.01-1.04-.2-1.55-.37-.63-.2-1.12-.31-1.08-.66.02-.18.27-.36.74-.55 2.92-1.27 4.86-2.11 5.83-2.51 2.78-1.16 3.35-1.36 3.73-1.37.08 0 .27.02.39.12.1.08.13.19.14.27-.01.06.01.24 0 .29z"/>
  </svg>
);

export const FolderIconWithShared = ({ shared }: { shared?: boolean }) => {
  return (
    <div style={{ position: 'relative', display: 'inline-flex', alignItems: 'center', justifyContent: 'center', width: '24px', height: '24px' }}>
      <IconFolder />
      {shared && (
        <div style={{
          position: 'absolute',
          bottom: '-3px',
          right: '-3px',
          backgroundColor: 'var(--md-sys-color-primary-container)',
          color: 'var(--md-sys-color-on-primary-container)',
          borderRadius: '50%',
          width: '12px',
          height: '12px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          boxShadow: 'var(--shadow-1)',
          border: '1px solid var(--md-sys-color-surface)'
        }} title="Shared folder">
          <svg width="8" height="8" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/>
          </svg>
        </div>
      )}
    </div>
  );
};

export const FileIcon = ({ name }: { name: string }) => {
  const ext = name.split('.').pop()?.toLowerCase() || '';

  // Return custom colorful icons based on extension
  if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico'].includes(ext)) {
    return (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="#00b4d8" style={{ flexShrink: 0 }}>
        <path d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0-2-.9-2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z"/>
      </svg>
    );
  }
  if (ext === 'pdf') {
    return (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="#e63946" style={{ flexShrink: 0 }}>
        <path d="M20 2H8c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm-8.5 7.5c0 .83-.67 1.5-1.5 1.5H9v2H7.5V7H10c.83 0 1.5.67 1.5 1.5v1zm5 2c0 .83-.67 1.5-1.5 1.5h-2.5V7H15c.83 0 1.5.67 1.5 1.5v3zm4-3.5H19v1h1.5V10H19v3h-1.5V7h3v1.5zM9 8.5h1v1H9v-1zm5 1.5h1v1.5h-1V10zM2 6v14c0 1.1.9 2 2 2h14v-2H4V6H2z"/>
      </svg>
    );
  }
  if (['doc', 'docx', 'txt', 'rtf'].includes(ext)) {
    return (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="#0077b6" style={{ flexShrink: 0 }}>
        <path d="M14 2H6c-1.1 0-1.99.9-1.99 2L4 20c0 1.1.89 2 1.99 2H18c1.1 0 2-.9 2-2V8l-6-6zm2 16H8v-2h8v2zm0-4H8v-2h8v2zm-3-5V3.5L18.5 9H13z"/>
      </svg>
    );
  }
  if (['xls', 'xlsx', 'csv'].includes(ext)) {
    return (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="#2a9d8f" style={{ flexShrink: 0 }}>
        <path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 2h1.5v3H12V5zm-2.5 0H11v3H9.5V5zM6 5h2v3H6V5zm12 14H6v-9h12v9z"/>
      </svg>
    );
  }
  if (['ppt', 'pptx'].includes(ext)) {
    return (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="#f4a261" style={{ flexShrink: 0 }}>
        <path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-2 10h-4v4h-2v-4H7v-2h10v2z"/>
      </svg>
    );
  }
  if (['zip', 'rar', '7z', 'tar', 'gz'].includes(ext)) {
    return (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="#9b5de5" style={{ flexShrink: 0 }}>
        <path d="M20 6h-8l-2-2H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm-6 3h-2v2h2v-2zm0 4h-2v2h2v-2z"/>
      </svg>
    );
  }
  if (['mp3', 'wav', 'flac', 'aac', 'm4a', 'ogg'].includes(ext)) {
    return (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="#0096c7" style={{ flexShrink: 0 }}>
        <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
      </svg>
    );
  }
  if (['mp4', 'webm', 'mov', 'mkv', 'avi'].includes(ext)) {
    return (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="#7209b7" style={{ flexShrink: 0 }}>
        <path d="M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z"/>
      </svg>
    );
  }
  if (['js', 'ts', 'jsx', 'tsx', 'html', 'css', 'go', 'py', 'java', 'xml', 'json', 'yaml', 'toml', 'cpp', 'c', 'sh'].includes(ext)) {
    return (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="#ff006e" style={{ flexShrink: 0 }}>
        <path d="M9.4 16.6L4.8 12l4.6-4.6L8 6l-6 6 6 6 1.4-1.4zm5.2 0l4.6-4.6-4.6-4.6L16 6l6 6-6 6-1.4-1.4z"/>
      </svg>
    );
  }
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="#8d99ae" style={{ flexShrink: 0 }}>
      <path d="M14 2H6c-1.1 0-1.99.9-1.99 2L4 20c0 1.1.89 2 1.99 2H18c1.1 0 2-.9 2-2V8l-6-6zm2 16H8v-2h8v2zm0-4H8v-2h8v2zm-3-5V3.5L18.5 9H13z"/>
    </svg>
  );
};
