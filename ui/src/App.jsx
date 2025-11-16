import { useState, useEffect } from 'react';
import Sidebar from './components/Sidebar';
import ChatInterface from './components/ChatInterface';
import ProcessPanel from './components/ProcessPanel';
import { sendMessage, sendMessageStream, getConversation, listConversations } from './services/api';
import './App.css';

function App() {
  const [conversations, setConversations] = useState([]);
  const [selectedConversationId, setSelectedConversationId] = useState(null);
  const [messages, setMessages] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isNewChat, setIsNewChat] = useState(false);

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
      
      // Auto-select first conversation if none selected and not starting a new chat
      if (!selectedConversationId && convs.length > 0 && !isNewChat) {
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
      // Generate a unique conversation ID for new chats
      let actualConversationId = conversationId;
      if (!actualConversationId && isNewChat) {
        actualConversationId = `conv-${Date.now()}-${Math.random().toString(36).substring(2, 11)}`;
      }

      // Use streaming for real-time updates
      await sendMessageStream(messageText, actualConversationId, (message) => {
        setMessages(prev => {
          // Remove temporary user message once we get the real one
          const withoutTemp = prev.filter(m => m.ID !== tempUserMessage.ID);
          
          // Check if this message already exists
          const existingIndex = withoutTemp.findIndex(m => m.ID === message.ID);
          if (existingIndex >= 0) {
            // Update existing message
            const updated = [...withoutTemp];
            updated[existingIndex] = message;
            return updated;
          } else {
            // Add new message
            return [...withoutTemp, message];
          }
        });
      });

      // Reload conversations to get updated list
      await loadConversations();
      
      // If this was a new chat or no conversation was selected, select the one that was created/used
      if (!conversationId || isNewChat) {
        const convs = await listConversations();
        if (convs.length > 0) {
          // Find the conversation that matches our ID, or get the last one
          const targetConv = actualConversationId 
            ? convs.find(c => c.id === actualConversationId) || convs[convs.length - 1]
            : convs[convs.length - 1];
          setSelectedConversationId(targetConv.id);
          setIsNewChat(false);
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
    setIsNewChat(true);
  };

  const handleSelectConversation = (conversationId) => {
    setSelectedConversationId(conversationId);
    setIsNewChat(false);
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
      <ProcessPanel />
    </div>
  );
}

export default App;
