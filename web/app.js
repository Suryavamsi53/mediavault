// API Base URL - Automatically detects if served by the Go server or opened standalone
const API_BASE = (window.location.port === '8080') ? '' : 'http://127.0.0.1:8080';

// App State
let mediaList = [];
let currentFilter = 'all';
let searchQuery = '';
let authToken = localStorage.getItem('mediavault_token') || '';
let currentUser = localStorage.getItem('mediavault_user') || '';
let authMode = 'login'; // 'login' or 'register'

// DOM Elements
const statusDot = document.getElementById('statusDot');
const statusText = document.getElementById('statusText');
const statTotal = document.getElementById('statTotal');
const statImages = document.getElementById('statImages');
const statVideos = document.getElementById('statVideos');
const statStorage = document.getElementById('statStorage');

const userPill = document.getElementById('userPill');
const userAvatar = document.getElementById('userAvatar');
const userNameDisplay = document.getElementById('userNameDisplay');
const logoutBtn = document.getElementById('logoutBtn');
const loginTriggerBtn = document.getElementById('loginTriggerBtn');

const authModal = document.getElementById('authModal');
const authModalTitle = document.getElementById('authModalTitle');
const tabAuthLogin = document.getElementById('tabAuthLogin');
const tabAuthRegister = document.getElementById('tabAuthRegister');
const authForm = document.getElementById('authForm');
const authUsername = document.getElementById('authUsername');
const authPassword = document.getElementById('authPassword');
const authErrorMsg = document.getElementById('authErrorMsg');
const authSubmitBtn = document.getElementById('authSubmitBtn');

const uploadDropzone = document.getElementById('uploadDropzone');
const fileInput = document.getElementById('fileInput');
const uploadProgressBox = document.getElementById('uploadProgressBox');
const uploadFileName = document.getElementById('uploadFileName');
const uploadPercent = document.getElementById('uploadPercent');
const uploadProgressFill = document.getElementById('uploadProgressFill');

const filterTabs = document.getElementById('filterTabs');
const searchInput = document.getElementById('searchInput');
const mediaGrid = document.getElementById('mediaGrid');

const mediaModal = document.getElementById('mediaModal');
const modalTitle = document.getElementById('modalTitle');
const modalMediaContainer = document.getElementById('modalMediaContainer');
const modalFileSize = document.getElementById('modalFileSize');
const modalFileType = document.getElementById('modalFileType');
const modalCloseBtn = document.getElementById('modalCloseBtn');
const modalCopyBtn = document.getElementById('modalCopyBtn');
const modalDownloadBtn = document.getElementById('modalDownloadBtn');
const toastContainer = document.getElementById('toastContainer');

let activeModalMedia = null;

// Initialize
document.addEventListener('DOMContentLoaded', () => {
  setupEventListeners();
  checkServerHealth();
  updateAuthUI();

  if (authToken) {
    fetchMedia();
  } else {
    openAuthModal('login');
  }

  // Periodic health check every 15 seconds
  setInterval(checkServerHealth, 15000);
});

function updateAuthUI() {
  if (authToken && currentUser) {
    userPill.style.display = 'flex';
    loginTriggerBtn.style.display = 'none';
    userNameDisplay.textContent = currentUser;
    userAvatar.textContent = currentUser.charAt(0).toUpperCase();
  } else {
    userPill.style.display = 'none';
    loginTriggerBtn.style.display = 'block';
  }
}

function openAuthModal(mode = 'login') {
  authMode = mode;
  authErrorMsg.style.display = 'none';
  authErrorMsg.textContent = '';
  authUsername.value = '';
  authPassword.value = '';

  if (authMode === 'login') {
    authModalTitle.textContent = 'Sign In to MediaVault';
    tabAuthLogin.classList.add('active');
    tabAuthRegister.classList.remove('active');
    authSubmitBtn.textContent = 'Sign In to My Vault';
  } else {
    authModalTitle.textContent = 'Create MediaVault Account';
    tabAuthRegister.classList.add('active');
    tabAuthLogin.classList.remove('active');
    authSubmitBtn.textContent = 'Create Account & Access Vault';
  }

  authModal.classList.add('open');
  if (typeof lenis !== 'undefined' && lenis) lenis.stop();
}

function closeAuthModal() {
  authModal.classList.remove('open');
  if (typeof lenis !== 'undefined' && lenis) lenis.start();
}

function setupEventListeners() {
  // Auth Controls
  tabAuthLogin.addEventListener('click', () => openAuthModal('login'));
  tabAuthRegister.addEventListener('click', () => openAuthModal('register'));
  loginTriggerBtn.addEventListener('click', () => openAuthModal('login'));

  logoutBtn.addEventListener('click', () => {
    localStorage.removeItem('mediavault_token');
    localStorage.removeItem('mediavault_user');
    authToken = '';
    currentUser = '';
    updateAuthUI();
    mediaList = [];
    updateStats();
    renderEmptyState('You are logged out. Please sign in to access your vault.');
    showToast('Logged out successfully.', 'info');
    openAuthModal('login');
  });

  authForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    authErrorMsg.style.display = 'none';
    authSubmitBtn.disabled = true;
    authSubmitBtn.textContent = 'Authenticating...';

    const username = authUsername.value.trim();
    const password = authPassword.value;
    const endpoint = authMode === 'login' ? '/v1/login' : '/v1/register';

    try {
      const res = await fetch(`${API_BASE}${endpoint}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.message || 'Authentication failed');
      }

      authToken = data.token;
      currentUser = data.username || username;
      localStorage.setItem('mediavault_token', authToken);
      localStorage.setItem('mediavault_user', currentUser);

      closeAuthModal();
      updateAuthUI();
      showToast(`Welcome back, ${currentUser}!`, 'success');
      fetchMedia();
    } catch (err) {
      authErrorMsg.textContent = err.message || 'An error occurred';
      authErrorMsg.style.display = 'block';
    } finally {
      authSubmitBtn.disabled = false;
      authSubmitBtn.textContent = authMode === 'login' ? 'Sign In to My Vault' : 'Create Account & Access Vault';
    }
  });

  // Drag & drop
  uploadDropzone.addEventListener('click', (e) => {
    if (e.target.closest('#uploadProgressBox')) return;
    if (!authToken) {
      openAuthModal('login');
      return;
    }
    fileInput.click();
  });

  uploadDropzone.addEventListener('dragover', (e) => {
    e.preventDefault();
    uploadDropzone.classList.add('dragover');
  });

  uploadDropzone.addEventListener('dragleave', () => {
    uploadDropzone.classList.remove('dragover');
  });

  uploadDropzone.addEventListener('drop', (e) => {
    e.preventDefault();
    uploadDropzone.classList.remove('dragover');
    if (!authToken) {
      openAuthModal('login');
      return;
    }
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      handleMultipleFiles(Array.from(e.dataTransfer.files));
    }
  });

  fileInput.addEventListener('change', (e) => {
    if (e.target.files && e.target.files.length > 0) {
      handleMultipleFiles(Array.from(e.target.files));
    }
  });

  // Filters
  filterTabs.addEventListener('click', (e) => {
    const btn = e.target.closest('.tab-btn');
    if (!btn) return;
    document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    currentFilter = btn.dataset.filter;
    renderMedia();
  });

  // Search
  searchInput.addEventListener('input', (e) => {
    searchQuery = e.target.value.toLowerCase().trim();
    renderMedia();
  });

  // Media Modal
  modalCloseBtn.addEventListener('click', closeModal);
  mediaModal.addEventListener('click', (e) => {
    if (e.target === mediaModal) closeModal();
  });

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && mediaModal.classList.contains('open')) {
      closeModal();
    }
  });

  modalCopyBtn.addEventListener('click', () => {
    if (!activeModalMedia) return;
    const url = getFullMediaUrl(activeModalMedia.download_url);
    copyToClipboard(url, 'Streaming URL copied to clipboard!');
  });
}

// Server Health Status
async function checkServerHealth() {
  try {
    const res = await fetch(`${API_BASE}/healthcheck`, { method: 'GET' });
    if (res.ok) {
      statusDot.className = 'status-dot online';
      statusText.textContent = 'PostgreSQL Connected';
    } else {
      throw new Error(`Status ${res.status}`);
    }
  } catch (err) {
    statusDot.className = 'status-dot offline';
    statusText.textContent = 'Server / DB Offline';
  }
}

// Fetch Media Records for the authenticated user
async function fetchMedia() {
  if (!authToken) {
    renderEmptyState('Please log in to view your files.');
    return;
  }

  try {
    const res = await fetch(`${API_BASE}/v1/media?per_page=1000`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });

    if (res.status === 401) {
      localStorage.removeItem('mediavault_token');
      authToken = '';
      updateAuthUI();
      openAuthModal('login');
      return;
    }

    if (!res.ok) throw new Error('Failed to load media');
    const data = await res.json();
    mediaList = data.items || [];
    updateStats();
    renderMedia();
  } catch (err) {
    console.error(err);
    renderEmptyState('Failed to connect to backend server. Make sure PostgreSQL is started and Go server is running.');
  }
}

// Stats Calculation
function updateStats() {
  const total = mediaList.length;
  let images = 0;
  let videos = 0;
  let totalBytes = 0;

  mediaList.forEach(item => {
    totalBytes += item.file_size || 0;
    if (item.media_type && item.media_type.startsWith('video/')) {
      videos++;
    } else {
      images++;
    }
  });

  statTotal.textContent = total;
  statImages.textContent = images;
  statVideos.textContent = videos;
  statStorage.textContent = formatBytes(totalBytes);
}

function getStreamUrl(downloadUrl) {
  const separator = downloadUrl.includes('?') ? '&' : '?';
  return `${API_BASE}${downloadUrl}${separator}token=${encodeURIComponent(authToken)}`;
}

// Render Gallery
function renderMedia() {
  const filtered = mediaList.filter(item => {
    const isVideo = item.media_type && item.media_type.startsWith('video/');
    const matchesFilter = 
      currentFilter === 'all' ||
      (currentFilter === 'images' && !isVideo) ||
      (currentFilter === 'videos' && isVideo);

    const matchesSearch = !searchQuery || item.file_name.toLowerCase().includes(searchQuery);

    return matchesFilter && matchesSearch;
  });

  if (filtered.length === 0) {
    renderEmptyState(searchQuery ? 'No media found matching your search.' : 'Your vault is empty. Upload your private images or videos above!');
    return;
  }

  mediaGrid.innerHTML = '';
  filtered.forEach(item => {
    const isVideo = item.media_type && item.media_type.startsWith('video/');
    const card = document.createElement('div');
    card.className = 'media-card';

    const streamUrl = getStreamUrl(item.download_url);

    card.innerHTML = `
      <div class="media-preview-container" data-id="${item.id}">
        ${isVideo ? `
          <div class="video-badge-pill">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
              <polygon points="5 3 19 12 5 21 5 3"/>
            </svg>
            Video
          </div>
          <div class="video-play-btn">
            <svg viewBox="0 0 24 24">
              <polygon points="5 3 19 12 5 21 5 3"/>
            </svg>
          </div>
          <video src="${streamUrl}#t=0.5" preload="metadata" muted playsinline style="width: 100%; height: 100%; object-fit: cover;"></video>
        ` : `
          <img src="${streamUrl}" alt="${escapeHtml(item.file_name)}" class="media-preview-img" loading="lazy" onerror="this.src='data:image/svg+xml;utf8,<svg xmlns=\\'http://www.w3.org/2000/svg\\' width=\\'100\\' height=\\'100\\'><rect fill=\\'%23121a2f\\' width=\\'100\\' height=\\'100\\'/><text fill=\\'%2364748b\\' x=\\'50%\\' y=\\'50%\\' text-anchor=\\'middle\\' dy=\\'.3em\\'>Image</text></svg>'">
        `}
      </div>
      <div class="media-card-body">
        <div class="media-name" title="${escapeHtml(item.file_name)}">${escapeHtml(item.file_name)}</div>
        <div class="media-meta-row">
          <span>${formatBytes(item.file_size)}</span>
          <span>${formatDate(item.created_at)}</span>
        </div>
        <div class="media-actions">
          <button class="btn-action btn-copy" data-url="${streamUrl}" title="Copy Link">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
            </svg>
            Link
          </button>
          <a class="btn-action" href="${streamUrl}" download="${escapeHtml(item.file_name)}" title="Download File">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/>
              <line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
            Save
          </a>
          <button class="btn-action btn-delete" data-id="${item.id}" title="Delete">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="3 6 5 6 21 6"/>
              <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
            </svg>
          </button>
        </div>
      </div>
    `;

    // Preview click -> open modal
    card.querySelector('.media-preview-container').addEventListener('click', () => {
      openModal(item);
    });

    // Copy URL
    card.querySelector('.btn-copy').addEventListener('click', (e) => {
      e.stopPropagation();
      const fullUrl = getFullMediaUrl(item.download_url);
      copyToClipboard(fullUrl, 'Stream URL copied to clipboard!');
    });

    // Delete
    card.querySelector('.btn-delete').addEventListener('click', async (e) => {
      e.stopPropagation();
      if (confirm(`Are you sure you want to delete "${item.file_name}" from your vault?`)) {
        await deleteMedia(item.id);
      }
    });

    mediaGrid.appendChild(card);
  });
}

function renderEmptyState(message) {
  mediaGrid.innerHTML = `
    <div class="empty-state">
      <div class="empty-icon">
        <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
          <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
          <circle cx="8.5" cy="8.5" r="1.5"/>
          <polyline points="21 15 16 10 5 21"/>
        </svg>
      </div>
      <h3 class="empty-title">Vault is Empty</h3>
      <p class="empty-desc">${escapeHtml(message)}</p>
    </div>
  `;
}

// Upload Multiple Files with Live Progress
async function handleMultipleFiles(files) {
  if (!files || files.length === 0) return;
  if (!authToken) {
    openAuthModal('login');
    return;
  }

  const maxBytes = 25 * 1024 * 1024; // 25 MB
  const validFiles = [];

  for (const file of files) {
    if (file.size > maxBytes) {
      showToast(`"${file.name}" exceeds 25 MB limit and was skipped.`, 'error');
    } else {
      validFiles.push(file);
    }
  }

  if (validFiles.length === 0) {
    fileInput.value = '';
    return;
  }

  uploadProgressBox.style.display = 'block';
  let successCount = 0;

  for (let i = 0; i < validFiles.length; i++) {
    const file = validFiles[i];
    uploadFileName.textContent = `Uploading file ${i + 1} of ${validFiles.length}: ${file.name} (${formatBytes(file.size)})`;

    try {
      await uploadSingleFile(file, (pct) => {
        const overallPct = Math.round(((i + pct / 100) / validFiles.length) * 100);
        uploadPercent.textContent = `${overallPct}%`;
        uploadProgressFill.style.width = `${overallPct}%`;
      });
      successCount++;
    } catch (err) {
      showToast(`Failed to upload "${file.name}": ${err.message || 'Error'}`, 'error');
    }
  }

  uploadPercent.textContent = '100%';
  uploadProgressFill.style.width = '100%';

  setTimeout(() => {
    uploadProgressBox.style.display = 'none';
    fileInput.value = '';
  }, 700);

  if (successCount > 0) {
    showToast(`Successfully uploaded ${successCount} file${successCount > 1 ? 's' : ''}!`, 'success');
    fetchMedia();
  }
}

function uploadSingleFile(file, onProgress) {
  return new Promise((resolve, reject) => {
    const formData = new FormData();
    formData.append('file', file);

    const xhr = new XMLHttpRequest();
    xhr.open('POST', `${API_BASE}/v1/media/upload`, true);
    if (authToken) {
      xhr.setRequestHeader('Authorization', `Bearer ${authToken}`);
    }

    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && onProgress) {
        const pct = Math.round((e.loaded / e.total) * 100);
        onProgress(pct);
      }
    };

    xhr.onload = () => {
      if (xhr.status === 201 || xhr.status === 200) {
        resolve();
      } else {
        let errMsg = 'Upload failed';
        try {
          const res = JSON.parse(xhr.responseText);
          errMsg = res.message || errMsg;
        } catch (_) {}
        reject(new Error(errMsg));
      }
    };

    xhr.onerror = () => {
      reject(new Error('Network error'));
    };

    xhr.send(formData);
  });
}

// Delete Media Item
async function deleteMedia(id) {
  try {
    const res = await fetch(`${API_BASE}/v1/media/${id}`, {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });

    if (res.ok) {
      showToast('Media deleted successfully.', 'success');
      fetchMedia();
    } else {
      throw new Error('Delete request failed');
    }
  } catch (err) {
    showToast('Failed to delete media from your vault.', 'error');
  }
}

// Open Lightbox / Theatre Modal
function openModal(item) {
  activeModalMedia = item;
  modalTitle.textContent = item.file_name;
  modalFileSize.textContent = formatBytes(item.file_size);
  modalFileType.textContent = item.media_type;

  const streamUrl = getStreamUrl(item.download_url);
  modalDownloadBtn.href = streamUrl;
  modalDownloadBtn.setAttribute('download', item.file_name);

  const isVideo = item.media_type && item.media_type.startsWith('video/');
  if (isVideo) {
    modalMediaContainer.innerHTML = `
      <video src="${streamUrl}" controls autoplay playsinline></video>
    `;
  } else {
    modalMediaContainer.innerHTML = `
      <img src="${streamUrl}" alt="${escapeHtml(item.file_name)}">
    `;
  }

  mediaModal.classList.add('open');
  if (typeof lenis !== 'undefined' && lenis) lenis.stop();
}

function closeModal() {
  mediaModal.classList.remove('open');
  if (typeof lenis !== 'undefined' && lenis) lenis.start();
  const video = modalMediaContainer.querySelector('video');
  if (video) {
    video.pause();
    video.src = '';
  }
  modalMediaContainer.innerHTML = '';
  activeModalMedia = null;
}

// Utilities
function formatBytes(bytes) {
  if (!bytes || bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function formatDate(dateStr) {
  if (!dateStr) return '';
  const date = new Date(dateStr);
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

function getFullMediaUrl(path) {
  const separator = path.includes('?') ? '&' : '?';
  const fullPath = `${path}${separator}token=${encodeURIComponent(authToken)}`;
  if (fullPath.startsWith('http')) return fullPath;
  const origin = API_BASE || window.location.origin;
  return `${origin}${fullPath}`;
}

function copyToClipboard(text, message) {
  navigator.clipboard.writeText(text).then(() => {
    showToast(message, 'success');
  }).catch(() => {
    const input = document.createElement('input');
    input.value = text;
    document.body.appendChild(input);
    input.select();
    document.execCommand('copy');
    document.body.removeChild(input);
    showToast(message, 'success');
  });
}

function showToast(message, type = 'info') {
  const toast = document.createElement('div');
  toast.className = `toast ${type}`;
  toast.innerHTML = `
    <span>${escapeHtml(message)}</span>
  `;
  toastContainer.appendChild(toast);

  setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transform = 'translateX(100%)';
    toast.style.transition = 'all 0.3s ease';
    setTimeout(() => toast.remove(), 300);
  }, 3500);
}

function escapeHtml(str) {
  if (!str) return '';
  return str.replace(/[&<>"']/g, m => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#039;'
  })[m]);
}

/* ========================================================
   Momentum Smooth Scroll Engine (Lenis)
   - Studio Freight Lenis smooth momentum scrolling
   - Natural inertia without scroll locking
   - Floating Scroll-to-Top button
   ======================================================== */
let lenis = null;
if (typeof Lenis !== 'undefined') {
  lenis = new Lenis({
    duration: 1.15,
    easing: (t) => Math.min(1, 1.001 - Math.pow(2, -10 * t)),
    orientation: 'vertical',
    gestureOrientation: 'vertical',
    smoothWheel: true,
    wheelMultiplier: 1,
    touchMultiplier: 1.2,
  });

  function raf(time) {
    lenis.raf(time);
    requestAnimationFrame(raf);
  }
  requestAnimationFrame(raf);
}

// Scroll to Top Button
const scrollTopBtn = document.getElementById('scrollTopBtn');
if (scrollTopBtn) {
  window.addEventListener('scroll', () => {
    if (window.scrollY > 300) {
      scrollTopBtn.classList.add('visible');
    } else {
      scrollTopBtn.classList.remove('visible');
    }
  }, { passive: true });

  scrollTopBtn.addEventListener('click', () => {
    if (lenis) {
      lenis.scrollTo(0);
    } else {
      window.scrollTo({ top: 0, behavior: 'smooth' });
    }
  });
}
