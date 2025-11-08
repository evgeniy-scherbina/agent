const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

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

