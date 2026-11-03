export interface SelectedPlace {
  id: string;
  label: string;
  latitude: number;
  longitude: number;
}
export type RideMode = "auto" | "partner" | "lift";
export type RideStatus =
  "matching" | "ready" | "travelling" | "completed" | "cancelled";
export interface Ride {
  fromPlace?: SelectedPlace;
  toPlace?: SelectedPlace;
  id: string;
  /** Assigned by matching; never chosen by the passenger. */
  mode: RideMode | null;
  from: string;
  to: string;
  date: string;
  time: string;
  seats: number;
  members: string[];
  host: string;
  status: RideStatus;
  womenOnly: boolean;
  allowLift?: boolean;
  poolId?: string;
  matchNote?: string;
  timeZone?: string;
  note: string;
  flexibility?: number;
  driver: boolean;
  confirmed: boolean;
  arrived: boolean;
}
export interface Message {
  id: string;
  group: string;
  author: string;
  text: string;
  time: string;
}
export interface Notice {
  id: string;
  title: string;
  text: string;
  read: boolean;
  path: string;
}
export interface Ticket {
  id: string;
  subject: string;
  category: string;
  status: "open" | "in progress" | "resolved";
  messages: Message[];
}
export interface SafetyReport {
  id: string;
  messages?: Message[];
  person: string;
  reason: string;
  details: string;
  status: "open" | "reviewed" | "dismissed";
  action?: string;
}
export interface Profile {
  name: string;
  email: string;
  phone: string;
  bio: string;
  pronouns: string;
  emergency: string;
  vehicle: string;
  plate: string;
  seats: number;
  womenOnlyEligible?: boolean;
}
export interface Person {
  name: string;
  email: string;
  photoUrl?: string;
  trips: number;
  rating: string;
  role: string;
  status: "active" | "suspended" | "banned";
}

/** A driver's standing route is separate from a passenger's journey request. */
export interface LiftOffer {
  fromPlace?: SelectedPlace;
  toPlace?: SelectedPlace;
  id: string;
  owner: string;
  from: string;
  to: string;
  date: string;
  time: string;
  flexibility: number;
  seats: number;
  vehicle: string;
  plate: string;
  womenOnly: boolean;
  alwaysActive: boolean;
  enabled: boolean;
  timeZone: string;
}
export interface PoolMember {
  name: string;
  womenOnlyEligible: boolean;
}
export interface PoolCandidate {
  id: string;
  kind: RideMode;
  from: string;
  to: string;
  date: string;
  time: string;
  flexibility: number;
  capacity: number;
  members: PoolMember[];
  womenOnly: boolean;
  offer?: LiftOffer;
  durationMinutes?: number;
}
