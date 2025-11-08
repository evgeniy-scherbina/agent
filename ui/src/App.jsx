import { useState, useEffect } from 'react';
import Sidebar from './components/Sidebar';
import ChatInterface from './components/ChatInterface';
import { sendMessage, getConversation, listConversations } from './services/api';
import './App.css';

function App() {
  const [conversations, setConversations] = useState([]);
  const [selectedConversationId, setSelectedConversationId] = useState(null);
  const [messages, setMessages] = useState([]);
  const [isLoading, setIsLoading] = useState(false);

  // Load conversations on mount
  useEffect(() => {
    loadConversations();
  }, []);

  // Load conversation messages when selection changes
  useEffect(() => {
    if (selectedConversationId) {
      loadConversation(selectedConversationId);
    } else {
      setMessages([]);
    }
  }, [selectedConversationId]);

  const loadConversations = async () => {
    try {
      const convs = await listConversations();
      setConversations(convs);
      
      // Auto-select first conversation if none selected
      if (!selectedConversationId && convs.length > 0) {
        setSelectedConversationId(convs[0].id);
      }
    } catch (error) {
      console.error('Failed to load conversations:', error);
    }
  };

  const loadConversation = async (conversationId) => {
    try {
      const conv = await getConversation(conversationId);
      setMessages(conv.messages || []);
    } catch (error) {
      console.error('Failed to load conversation:', error);
    }
  };

  const handleSendMessage = async (messageText, conversationId) => {
    // Create a temporary user message to show immediately (optimistic update)
    const tempUserMessage = {
      ID: `temp-${Date.now()}`,
      role: 'user',
      content: messageText,
    };

    // Add user message immediately to the UI
    setMessages(prev => [...prev, tempUserMessage]);
    setIsLoading(true);

    try {
      const response = await sendMessage(messageText, conversationId);
      
      // Replace temporary message and add assistant response with actual messages from API
      setMessages(prev => {
        // Remove the temporary message
        const withoutTemp = prev.filter(m => m.ID !== tempUserMessage.ID);
        
        // Add the actual messages from the API response
        // The API returns both user and assistant messages, so we need to merge them
        const existingIds = new Set(withoutTemp.map(m => m.ID));
        const newMessages = response.messages.filter(m => !existingIds.has(m.ID));
        
        return [...withoutTemp, ...newMessages];
      });

      // Reload conversations to get updated list
      await loadConversations();
      
      // If no conversation was selected, select the one that was created/used
      if (!conversationId) {
        const convs = await listConversations();
        if (convs.length > 0) {
          setSelectedConversationId(convs[convs.length - 1].id);
        }
      }
    } catch (error) {
      console.error('Failed to send message:', error);
      // Remove the temporary message on error
      setMessages(prev => prev.filter(m => m.ID !== tempUserMessage.ID));
      alert('Failed to send message. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  const handleNewConversation = () => {
    setSelectedConversationId(null);
    setMessages([]);
  };

  const handleSelectConversation = (conversationId) => {
    setSelectedConversationId(conversationId);
  };

  const handleDeleteConversation = async (conversationId) => {
    if (!confirm('Are you sure you want to delete this conversation?')) {
      return;
    }

    // Remove from local state
    setConversations(prev => prev.filter(conv => conv.id !== conversationId));
    
    // If deleted conversation was selected, clear selection
    if (selectedConversationId === conversationId) {
      setSelectedConversationId(null);
      setMessages([]);
    }

    // Note: This is client-side only. To persist deletion, you'd need a DELETE endpoint
    // For now, the conversation will reappear on page refresh since it's still on the server
  };

  return (
    <div className="app">
      <Sidebar
        conversations={conversations}
        selectedConversationId={selectedConversationId}
        onSelectConversation={handleSelectConversation}
        onNewConversation={handleNewConversation}
        onDeleteConversation={handleDeleteConversation}
      />
      <ChatInterface
        conversationId={selectedConversationId}
        messages={messages}
        onSendMessage={handleSendMessage}
        isLoading={isLoading}
      />
    </div>
  );
}

export default App;
