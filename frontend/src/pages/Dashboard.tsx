import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import toast from 'react-hot-toast';

interface File {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  DeletedAt: string | null;
  user_id: number;
  filename: string;
  original_name: string;
  size: number;
  mime_type: string;
  share_url?: string;
}

declare global {
  interface Window {
    env: {
      REACT_APP_API_URL?: string;
    };
  }
}

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

const Dashboard = () => {
  const [files, setFiles] = useState<File[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const token = localStorage.getItem('token');

  const fetchFiles = async () => {
    try {
      const response = await axios.get(`${API_URL}/api/files`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (response.data.files) {
        setFiles(response.data.files);
      }
    } catch (error) {
      console.error('Error fetching files:', error);
      toast.error('Failed to fetch files');
    }
  };

  const memoizedFetchFiles = useCallback(fetchFiles, [token]);

  useEffect(() => {
    if (token) {
      memoizedFetchFiles();
    }
  }, [token, memoizedFetchFiles]);

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    if (!e.target.files?.length) return;

    const file = e.target.files[0];
    if (file.size > 10 * 1024 * 1024) { // 10MB limit
      toast.error('File size exceeds 10MB limit');
      return;
    }

    const formData = new FormData();
    formData.append('file', file);

    try {
      console.log('Uploading file to:', `${API_URL}/api/files/upload`);
      const response = await axios.post(`${API_URL}/api/files/upload`, formData, {
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'multipart/form-data',
        },
        onUploadProgress: (progressEvent) => {
          const percentCompleted = Math.round((progressEvent.loaded * 100) / progressEvent.total!);
          toast.loading(`Uploading: ${percentCompleted}%`, { id: 'upload-progress' });
        },
      });
      
      if (response.data.file) {
        setFiles(prevFiles => [...prevFiles, response.data.file]);
        toast.success('File uploaded successfully', { id: 'upload-progress' });
        // Refresh the file list after successful upload
        memoizedFetchFiles();
      } else {
        toast.error('Invalid response from server', { id: 'upload-progress' });
      }
    } catch (error) {
      console.error('Error uploading file:', error);
      if (axios.isAxiosError(error)) {
        if (error.response?.status === 413) {
          toast.error('File size exceeds server limit', { id: 'upload-progress' });
        } else if (error.response?.data?.error) {
          toast.error(error.response.data.error, { id: 'upload-progress' });
        } else {
          toast.error('Failed to upload file', { id: 'upload-progress' });
        }
      } else {
        toast.error('An unexpected error occurred', { id: 'upload-progress' });
      }
    }
  };

  const handleDelete = async (fileId: number) => {
    if (!fileId || isNaN(fileId)) {
      console.error('Invalid file ID:', fileId);
      toast.error('Invalid file ID');
      return;
    }

    try {
      console.log('Attempting to delete file:', fileId);
      const response = await axios.delete(`${API_URL}/api/files/${fileId}`, {
        headers: { 
          Authorization: `Bearer ${token}`,
        },
      });
      
      if (response.status === 200) {
        setFiles(prevFiles => prevFiles.filter(file => file.ID !== fileId));
        toast.success('File deleted successfully');
        // Refresh the file list after successful deletion
        memoizedFetchFiles();
      }
    } catch (error) {
      console.error('Delete error:', error);
      toast.error('Failed to delete file');
    }
  };

  const handleShare = async (fileId: number) => {
    if (!fileId || isNaN(fileId)) {
      console.error('Invalid file ID:', fileId);
      toast.error('Invalid file ID');
      return;
    }

    try {
      console.log('Attempting to share file:', fileId);
      const response = await axios.get(`${API_URL}/api/files/share/${fileId}`, {
        headers: { 
          Authorization: `Bearer ${token}`,
        },
      });
      
      const { share_url: shareUrl, file: updatedFile } = response.data;
      if (!shareUrl) {
        toast.error('Invalid share URL received from server');
        return;
      }

      // Copy to clipboard
      await navigator.clipboard.writeText(shareUrl);
      
      setFiles(prevFiles => prevFiles.map(file => 
        file.ID === fileId ? { ...file, ...updatedFile, share_url: shareUrl } : file
      ));
      
      toast.success('Share URL copied to clipboard!');
    } catch (error) {
      console.error('Share error:', error);
      toast.error('Failed to generate share URL');
    }
  };

  const handleSearch = async () => {
    try {
      const response = await axios.get(`${API_URL}/api/files/search?query=${searchQuery}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      setFiles(response.data.files);
    } catch (error) {
      toast.error('Failed to search files');
    }
  };

  const formatDate = (dateString: string) => {
    if (!dateString) return 'N/A';
    try {
      const date = new Date(dateString);
      if (isNaN(date.getTime())) return 'N/A';
      return date.toLocaleDateString();
    } catch (error) {
      return 'N/A';
    }
  };

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-2xl font-bold">My Files</h1>
        <div className="flex gap-4">
          <div className="flex">
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search files..."
              className="px-4 py-2 border border-gray-300 rounded-l-md focus:outline-none focus:ring-primary-500 focus:border-primary-500"
            />
            <button
              onClick={handleSearch}
              className="px-4 py-2 bg-primary-600 text-white rounded-r-md hover:bg-primary-700"
            >
              Search
            </button>
          </div>
          <label className="cursor-pointer bg-primary-600 text-white px-4 py-2 rounded-md hover:bg-primary-700">
            Upload File
            <input 
              type="file" 
              className="hidden" 
              onChange={handleFileUpload}
              accept="*/*"
            />
          </label>
        </div>
      </div>

      {files.length === 0 ? (
        <div className="text-center py-8">
          <p className="text-gray-500">No files uploaded yet. Click "Upload File" to get started.</p>
        </div>
      ) : (
        <div className="bg-white shadow rounded-lg overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Size</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Uploaded</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {files.map((file) => (
                <tr key={file.ID}>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm font-medium text-gray-900">{file.original_name}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm text-gray-500">{Math.round(file.size / 1024)} KB</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm text-gray-500">{file.mime_type}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm text-gray-500">{formatDate(file.UpdatedAt)}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <button
                      onClick={() => {
                        console.log('Share button clicked for file:', file);
                        if (file && file.ID) {
                          handleShare(file.ID);
                        } else {
                          console.error('Invalid file object:', file);
                          toast.error('Invalid file data');
                        }
                      }}
                      className="text-primary-600 hover:text-primary-900 mr-4"
                    >
                      Share
                    </button>
                    <button
                      onClick={() => {
                        console.log('Delete button clicked for file:', file);
                        if (file && file.ID) {
                          if (window.confirm('Are you sure you want to delete this file?')) {
                            handleDelete(file.ID);
                          }
                        } else {
                          console.error('Invalid file object:', file);
                          toast.error('Invalid file data');
                        }
                      }}
                      className="text-red-600 hover:text-red-900"
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};

export default Dashboard; 