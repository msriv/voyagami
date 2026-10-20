# Design board index

Boards live in `design/boards/*.dc.html`. Live canvas: https://claude.ai/artifact/XxTE1B2QoZeDRXQGpQYrGW

How to read a board: each is one screen (390x844) or one spec page. They are reference for layout, copy, states and behaviour. Open one in a browser (needs `support.js` from the canvas export) or read the HTML directly. Ticket numbers are in Linear team VOY.

Sample data (Bengaluru persona, ₹, UPI, Hanoi trip) is illustrative only.

## Brand and navigation
| File | What it is | Tickets |
|---|---|---|
| Brand | Brand sheet: colours, type, motifs, logo (seal + wordmark, app icon) | VOY-11, VOY-12 |
| Main | Home | VOY-11, VOY-22 |
| HomeMulti, HomeDark | Home with several trips; dark variant | VOY-22, VOY-11 |
| Explore | Explore tab | VOY-45 |
| Profile | Profile and settings | VOY-59 |

## Onboarding and tour
| File | What it is | Tickets |
|---|---|---|
| OnbWelcome, OnbSignIn, OnbConnect, OnbStyle, OnbEmpty | First-run flow | VOY-17, VOY-13, VOY-18 |
| Trial, TrialEnd | Trial offer and trial end | VOY-19, VOY-55 |
| Tut1 to Tut5, TutStamps | Five-step tour and first five stamps | VOY-20, VOY-21 |

## Trip planning
| File | What it is | Tickets |
|---|---|---|
| NewTrip | Create a trip | VOY-22 |
| Trip | Trip overview | VOY-23 |
| Itinerary | Day plan, reorder | VOY-24, VOY-26 |
| AddPlace | Search / paste link / manual | VOY-25 |
| LiveTrip | Day-of view | VOY-24, VOY-33 |
| Tasks | Checklist | VOY-23 |
| ImportBookings, ImportError, BookingReview | Bring in bookings | VOY-28 |

## Crew
| File | What it is | Tickets |
|---|---|---|
| InviteCrew | Invite flow | VOY-29 |
| GroupHub | Crew and decisions | VOY-31, VOY-32 |
| Poll | Polls | VOY-32 |
| Comments | Comments on places | VOY-30 |

## Money
| File | What it is | Tickets |
|---|---|---|
| AddExpense | Add expense | VOY-34, VOY-35, VOY-36 |
| Budget, TripBudget | Budget and settle-up | VOY-37, VOY-38 |

## Journal and recap
| File | What it is | Tickets |
|---|---|---|
| Journal | Journal day | VOY-39 |
| Recap, RecapPlayer | Recap screens and player | VOY-41, VOY-42 |

## Recommendations and bookings
| File | What it is | Tickets |
|---|---|---|
| RecsForYou, PlaceDetail | Picks and place detail | VOY-44, VOY-45 |
| RecsStays | Stays with partner hand-off | VOY-46 |
| RecsEvents | Experiences | VOY-47 |
| AskJaunt | Ask (AI trip drafts). File name predates the rename; the product is Voyagami | VOY-48 |
| RecsSpec | Data sources, ranking, ads decision, bookings and commission | VOY-43, VOY-44, VOY-46, VOY-49 |

## Plans and paywall
| File | What it is | Tickets |
|---|---|---|
| Plans | Plans screen | VOY-52 |
| PayRoute, PayCurrency, OfflineGate | Paywall moments | VOY-53 |
| ManageSub | Manage subscription | VOY-56 |
| PlansSpec | Limits, pricing, playbook, revenue scenarios | VOY-51, VOY-54, VOY-57 |

## Offline, notifications, locale
| File | What it is | Tickets |
|---|---|---|
| OfflineSync | Offline and sync states | VOY-62, VOY-63, VOY-64 |
| Notifications | Notification settings and examples | VOY-66, VOY-67 |
| LocalePreview, LocaleSpec | Region preview and spec | VOY-18, VOY-58, VOY-59, VOY-61 |

## Specs, flows and reviews
| File | What it is | Tickets |
|---|---|---|
| TechStack, StackCosts, SysArch | Stack, costs, architecture | VOY-5 to VOY-16, VOY-65 |
| DataMap | Where each piece of data comes from | VOY-10, VOY-72 |
| FlowOnboarding, FlowPlaces, FlowCrew, FlowExpenses, FlowBookings, FlowJournal | Data-acquisition flows | matching feature tickets |
| Evaluation | Design review and final pass findings | VOY-69 |

## Known design gaps (build correctly, do not copy)
- RecapPlayer photo card is about 14px taller than its frame in the mock.
- A few photos are low resolution placeholders (Wayanad, Coorg, Ha Long sunrise); real content comes from the places pipeline.
- `[CODE]`, `[AIRLINE + NO.]`, `[your-handle]` are fields the user fills in.
- Dark mode is only sketched on HomeDark.
