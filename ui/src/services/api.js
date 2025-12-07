// Use Coder dev URL or env var, fallback to relative path
const API_BASE_URL = import.meta.env.VITE_API_URL || 'https://8080--dev--agent--yevhenii--apps.dev.coder.com';

/**
 * Send a message to the agent
 * @param {string} message - The message to send
 * @param {string} conversationId - Optional conversation ID
 * @returns {Promise<{messages: Array}>}
 */
export const sendMessage = async (message, conversationId = null) => {
  const response = await fetch(`${API_BASE_URL}/api/chat`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      message,
      conversationId: conversationId || undefined,
    }),
  });

  if (!response.ok) {
    throw new Error(`Failed to send message: ${response.statusText}`);
  }

  return response.json();
};

/**
 * Send a message to the agent with streaming updates
 * @param {string} message - The message to send
 * @param {string} conversationId - Optional conversation ID
 * @param {Function} onMessage - Callback for each message update
 * @returns {Promise<void>}
 */
export const sendMessageStream = async (message, conversationId = null, onMessage) => {
  const response = await fetch(`${API_BASE_URL}/api/chat/stream`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      message,
      conversationId: conversationId || undefined,
    }),
  });

  if (!response.ok) {
    throw new Error(`Failed to send message: ${response.statusText}`);
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop(); // Keep incomplete line in buffer

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const data = line.slice(6); // Remove 'data: ' prefix
          try {
            const parsed = JSON.parse(data);
            if (parsed.type === 'connected' || parsed.type === 'keepalive') {
              continue;
            }
            if (parsed.type === 'done') {
              return;
            }
            if (parsed.type === 'error') {
              throw new Error(parsed.error);
            }
            // It's a message object
            if (onMessage) {
              onMessage(parsed);
            }
          } catch (e) {
            // Skip invalid JSON (like keepalive comments)
            if (data.trim() && !data.startsWith(':')) {
              console.warn('Failed to parse SSE data:', data, e);
            }
          }
        }
      }
    }
  } finally {
    reader.releaseLock();
  }
};

/**
 * Get a specific conversation by ID
 * @param {string} conversationId - The conversation ID
 * @returns {Promise<{id: string, messages: Array}>}
 */
export const getConversation = async (conversationId) => {
  const response = await fetch(`${API_BASE_URL}/api/conversations/${conversationId}`);

  if (!response.ok) {
    throw new Error(`Failed to get conversation: ${response.statusText}`);
  }

  return response.json();
};

/**
 * List all conversations
 * @returns {Promise<Array<{id: string, messages: Array}>>}
 */
export const listConversations = async () => {
  const response = await fetch(`${API_BASE_URL}/api/conversations`);

  if (!response.ok) {
    throw new Error(`Failed to list conversations: ${response.statusText}`);
  }

  return response.json();
};

/**
 * List all running background processes
 * @returns {Promise<Array<{pid: number, command: string, start_time: string, conversation_id: string}>>}
 */
export const listProcesses = async () => {
  const response = await fetch(`${API_BASE_URL}/api/processes`);

  if (!response.ok) {
    throw new Error(`Failed to list processes: ${response.statusText}`);
  }

  return response.json();
};

/**
 * Kill a background process by PID
 * @param {number} pid - The process ID to kill
 * @returns {Promise<{success: boolean, message: string}>}
 */
export const killProcess = async (pid) => {
  const response = await fetch(`${API_BASE_URL}/api/processes/${pid}/kill`, {
    method: 'POST',
  });

  if (!response.ok) {
    throw new Error(`Failed to kill process: ${response.statusText}`);
  }

  return response.json();
};

