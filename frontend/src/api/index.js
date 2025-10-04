import axios from 'axios';

const client = axios.create({
  baseURL: '/api',
  timeout: 10000,
  withCredentials: true
});

export const getCA = () => client.get('/ca');
export const generateCA = (payload) => client.post('/ca', payload);
export const downloadCACert = () => client.get('/ca/certificate', { responseType: 'blob' });

export const getSettings = () => client.get('/settings');
export const updateSettings = (payload) => client.put('/settings', payload);

export const listTemplates = () => client.get('/templates');
export const upsertTemplate = (payload) => client.post('/templates', payload);
export const deleteTemplate = (id) => client.delete(`/templates/${id}`);

export const listNodes = () => client.get('/nodes');
export const createNode = (payload) => client.post('/nodes', payload);
export const getNodeArtifacts = (id) => client.get(`/nodes/${id}/artifacts`);
export const getNodeConfig = (id) => client.get(`/nodes/${id}/config`, { responseType: 'blob' });
export const downloadNodeBundle = (id) => client.get(`/nodes/${id}/bundle`, { responseType: 'blob' });
export const getInstallScript = (id) => client.get(`/nodes/${id}/install-script`, { responseType: 'blob' });
export const getNodeNetwork = (id, range) => client.get(`/nodes/${id}/network`, { params: range ? { range } : {} });
export const submitNodeNetworkSamples = (id, payload) => client.post(`/nodes/${id}/network/samples`, payload);
export const getNodeNetworkTargets = (id) => client.get(`/nodes/${id}/network/targets`);
export const getPublicStatus = () => client.get('/public/status');
export const getPublicNodeNetwork = (id, range) => client.get(`/public/nodes/${id}/network`, { params: range ? { range } : {} });
export const deleteNode = (id) => client.delete(`/nodes/${id}`);
export const login = (payload) => client.post('/login', payload);
export const logout = () => client.post('/logout');
export const getProfile = () => client.get('/me');

export default client;
