import React, { useState, useRef, useEffect } from 'react';
import { Play, Pause, Volume2, ArrowLeft, VolumeX, Music, Headphones, Shuffle, SkipBack, SkipForward, Repeat } from 'lucide-react';
import ReactHowler from 'react-howler';
import { Cloudinary } from '@cloudinary/url-gen';
import axios from 'axios';
import { Howl } from 'howler';

// Initialize Cloudinary with your cloud name and API key
const CLOUDINARY_CLOUD_NAME = 'drenighdk';
// Note: We're not using the API secret in the frontend for security reasons

const cld = new Cloudinary({
  cloud: {
    cloudName: CLOUDINARY_CLOUD_NAME
  }
});

// Helper function to format date from filename
const formatDateFromFilename = (filename) => {
  try {
    // Extract the date part (assuming format: yyyy-mm-dd.mp3)
    const datePart = filename.split('.')[0];
    
    // Parse the date components
    const [year, month, day] = datePart.split('-').map(num => parseInt(num, 10));
    
    // Create a formatted date string
    const date = new Date(year, month - 1, day); // month is 0-indexed in JS Date
    
    // Format options
    const options = { year: 'numeric', month: 'long', day: 'numeric' };
    return date.toLocaleDateString('en-US', options);
  } catch (error) {
    console.error('Error formatting date:', error);
    return 'Unknown Date';
  }
};

// Helper function to format remake title from filename
const formatRemakeTitle = (filename) => {
  try {
    // Remove the extension and "cover-" prefix
    const titlePart = filename.split('.')[0].replace('cover-', '');
    
    // Split into song and artist
    const parts = titlePart.split('-');
    if (parts.length < 2) return titlePart; // Fallback if format doesn't match
    
    // Format the song name and artist
    // First replace triple underscores with a temporary token, then handle single underscores, then restore &
    const formatText = (text) => {
      return text
        .replace(/___/g, '{{AMP}}')  // Replace ___ with temporary token
        .replace(/_/g, ' ')          // Replace _ with space
        .replace(/{{AMP}}/g, ' & '); // Replace temporary token with &
    };
    
    const songName = formatText(parts[0]);
    const artist = formatText(parts[1]);
    
    return `${songName} (${artist})`;
  } catch (error) {
    console.error('Error formatting remake title:', error);
    return 'Unknown Track';
  }
};

// Helper function to format original title from filename
const formatOriginalTitle = (filename) => {
  try {
    // Remove the extension
    const titlePart = filename.split('.')[0];
    
    // Format the text using the same approach as remakes
    const formatText = (text) => {
      return text
        .replace(/___/g, '{{AMP}}')  // Replace ___ with temporary token
        .replace(/_/g, ' ')          // Replace _ with space
        .replace(/{{AMP}}/g, ' & '); // Replace temporary token with &
    };
    
    return formatText(titlePart);
  } catch (error) {
    console.error('Error formatting original title:', error);
    return 'Unknown Track';
  }
};

const formatTime = (timeInSeconds) => {
  if (!timeInSeconds || isNaN(timeInSeconds)) return "00:00";
  const minutes = Math.floor(timeInSeconds / 60);
  const seconds = Math.floor(timeInSeconds % 60);
  return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
};

const HomeScreen = ({ onNavigate, trackCounts }) => (
  <div className="flex flex-col h-full">
    <div className="sticky top-0 bg-white z-10 p-4 md:p-8 border-b">
      <div className="flex justify-end items-center">
        <h1 className="text-2xl md:text-4xl font-bold">
          Kien's Homemade Music
        </h1>
      </div>
    </div>
    
    <div className="flex-1 overflow-y-auto p-4 md:p-8 flex items-center">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 md:gap-8 w-full">
        <div 
          className="flex flex-col items-start cursor-pointer transform transition-all duration-300 hover:scale-105" 
          onClick={() => onNavigate('dailySessions')}
        >
          <div className="w-full aspect-square rounded-lg shadow-xl overflow-hidden mb-4">
            <img 
              src="/images/daily-sessions-cover.jpeg" 
              alt="Daily Sessions"
              className="w-full h-full object-cover"
            />
          </div>
          <h2 className="text-xl md:text-2xl font-bold text-left">Daily Sessions</h2>
          <p className="text-base md:text-lg text-left">{trackCounts.dailySessions} Tracks</p>
        </div>
        
        <div 
          className="flex flex-col items-start cursor-pointer transform transition-all duration-300 hover:scale-105"
          onClick={() => onNavigate('remakes')}
        >
          <div className="w-full aspect-square rounded-lg shadow-xl overflow-hidden mb-4">
            <img 
              src="/images/remakes-cover.png" 
              alt="Remakes"
              className="w-full h-full object-cover"
            />
          </div>
          <h2 className="text-xl md:text-2xl font-bold text-left">Remakes</h2>
          <p className="text-base md:text-lg text-left">{trackCounts.remakes} Tracks</p>
        </div>
        
        <div 
          className="flex flex-col items-start cursor-pointer md:col-span-2 lg:col-span-1 transform transition-all duration-300 hover:scale-105"
          onClick={() => onNavigate('originals')}
        >
          <div className="w-full aspect-square rounded-lg shadow-xl overflow-hidden mb-4">
            <img 
              src="/images/originals-cover.png" 
              alt="Full Originals"
              className="w-full h-full object-cover"
            />
          </div>
          <h2 className="text-xl md:text-2xl font-bold text-left">Full Originals</h2>
          <p className="text-base md:text-lg text-left">{trackCounts.originals > 0 ? `${trackCounts.originals} Tracks` : "Work in progress, just wait"}</p>
        </div>
      </div>
    </div>
  </div>
);

const PlayerControls = ({ 
  isPlaying, 
  onPlayPause, 
  currentTime, 
  duration, 
  onSeek, 
  onSeekStart,
  onSeekEnd,
  volume, 
  isMuted, 
  onVolumeChange, 
  onMuteToggle,
  isSeeking,
  currentTrack,
  isDailySession,
  onPrevTrack,
  onNextTrack,
  isShuffled,
  onShuffleToggle,
  isRepeating,
  onRepeatToggle
}) => (
  <div className="p-4 md:p-8 flex flex-col border-t border-gray-300 bg-white space-y-4">
    {currentTrack && (
      <div className="text-center mb-2 md:mb-4 w-full">
        <h3 className="text-lg md:text-2xl font-bold text-orange-600 truncate px-4">
          {isDailySession ? currentTrack.date : currentTrack.title}
        </h3>
      </div>
    )}
    
    {/* Progress Bar */}
    <div className="flex items-center space-x-2">
      <span className="w-12 text-sm md:text-base text-center">{formatTime(currentTime)}</span>
      <input 
        type="range" 
        min="0" 
        max={duration || 100} 
        value={currentTime || 0}
        onChange={onSeek}
        onMouseDown={onSeekStart}
        onMouseUp={onSeekEnd}
        onTouchStart={onSeekStart}
        onTouchEnd={onSeekEnd}
        className={`flex-grow h-2 cursor-pointer ${isSeeking ? 'seeking' : ''}`}
        step="0.1"
      />
      <span className="w-12 text-sm md:text-base text-center">{formatTime(duration)}</span>
    </div>
    
    {/* Controls Container */}
    <div className="flex justify-between items-center px-4">
      {/* Empty div for spacing */}
      <div className="hidden md:block w-24"></div>
      
      {/* Centered Playback Controls */}
      <div className="flex items-center justify-center space-x-4 md:space-x-6">
        <button 
          onClick={onShuffleToggle}
          className={`cursor-pointer hover:opacity-80 active:opacity-60 ${isShuffled ? 'text-orange-500' : 'text-gray-400'}`}
        >
          <Shuffle className="w-5 h-5 md:w-6 md:h-6" />
        </button>
        
        <button 
          onClick={onPrevTrack}
          className="cursor-pointer hover:opacity-80 active:opacity-60"
        >
          <SkipBack className="w-7 h-7 md:w-8 md:h-8" />
        </button>
        
        <button 
          onClick={onPlayPause} 
          className="flex-shrink-0 cursor-pointer hover:opacity-80 active:opacity-60"
        >
          {isPlaying ? (
            <Pause className="w-10 h-10 md:w-12 md:h-12 text-orange-500" />
          ) : (
            <Play className="w-10 h-10 md:w-12 md:h-12 text-orange-500" />
          )}
        </button>
        
        <button 
          onClick={onNextTrack}
          className="cursor-pointer hover:opacity-80 active:opacity-60"
        >
          <SkipForward className="w-7 h-7 md:w-8 md:h-8" />
        </button>
        
        <button 
          onClick={onRepeatToggle}
          className={`cursor-pointer hover:opacity-80 active:opacity-60 ${isRepeating ? 'text-orange-500' : 'text-gray-400'}`}
        >
          <Repeat className="w-5 h-5 md:w-6 md:h-6" />
        </button>
      </div>

      {/* Right-aligned Volume Controls */}
      <div className="hidden md:flex items-center space-x-2 w-32">
        <button 
          onClick={onMuteToggle} 
          className="flex-shrink-0 cursor-pointer hover:opacity-80 active:opacity-60"
        >
          {isMuted ? (
            <VolumeX className="w-5 h-5" />
          ) : (
            <Volume2 className="w-5 h-5" />
          )}
        </button>
        <input 
          type="range"
          min="0"
          max="1"
          step="0.01"
          value={volume}
          onChange={onVolumeChange}
          className="w-24 h-1.5 cursor-pointer accent-orange-500"
        />
      </div>
    </div>
  </div>
);

const TrackList = ({ tracks, currentTrackIndex, isPlaying, onTrackSelect, isDailySession }) => (
  <div className="w-full md:w-2/3 flex flex-col overflow-y-auto">
    {tracks.map((track, index) => (
      <div 
        key={track.id} 
        className={`border-b border-gray-300 py-4 md:py-8 px-4 md:px-8 flex justify-between items-center cursor-pointer hover:bg-orange-100 ${currentTrackIndex === index ? 'bg-orange-50' : ''}`}
        onClick={() => onTrackSelect(index)}
      >
        <div className="flex-grow text-left">
          {isDailySession ? (
            <p className="text-base md:text-xl font-medium text-left truncate">{track.date}</p>
          ) : (
            <p className="text-base md:text-xl font-medium text-left truncate">{track.title}</p>
          )}
        </div>
        <div className="ml-4">
          {currentTrackIndex === index && isPlaying ? (
            <Pause className="w-8 h-8 md:w-10 md:h-10 text-orange-500" />
          ) : (
            <Play className="w-8 h-8 md:w-10 md:h-10" />
          )}
        </div>
      </div>
    ))}
  </div>
);

const CategoryScreen = ({ 
  onNavigateBack, 
  tracks, 
  currentTrackIndex, 
  isPlaying, 
  onTrackSelect,
  playerControls,
  audioError,
  categoryTitle,
  isDailySession = false,
  currentScreen
}) => (
  <div className="flex flex-col h-full">
    <div className="flex flex-col md:flex-row flex-1 overflow-hidden">
      <div className="w-full md:w-1/3 p-4 md:p-8 border-b md:border-b-0 md:border-r border-gray-300 flex flex-col items-center">
        <button 
          className="mb-4 md:mb-8 flex items-center text-orange-500 cursor-pointer hover:opacity-80 active:opacity-60 self-start" 
          onClick={onNavigateBack}
        >
          <ArrowLeft className="w-6 h-6 md:w-8 md:h-8 mr-2" />
          <span className="text-lg md:text-xl">Back</span>
        </button>
        <div className="w-48 md:w-full max-w-sm aspect-square rounded-lg shadow-xl overflow-hidden mb-4 md:mb-8">
          <img 
            src={`/images/${currentScreen === 'dailySessions' ? 'daily-sessions-cover.jpeg' : currentScreen === 'remakes' ? 'remakes-cover.png' : 'originals-cover.png'}`}
            alt={categoryTitle}
            className="w-full h-full object-cover"
          />
        </div>
        <h2 className="text-2xl md:text-3xl font-bold text-center">{categoryTitle}</h2>
        <p className="text-lg md:text-xl text-center">{tracks.length} Tracks</p>
      </div>
      
      <TrackList 
        tracks={tracks} 
        currentTrackIndex={currentTrackIndex} 
        isPlaying={isPlaying} 
        onTrackSelect={onTrackSelect}
        isDailySession={isDailySession}
      />
    </div>
    
    <div className="mt-auto">
      {playerControls}
      
      {audioError && (
        <div className="p-4 text-base md:text-xl text-red-500 text-center border-t border-red-200 bg-red-50">
          Error loading audio. Please check your connection or try another track.
        </div>
      )}
    </div>
  </div>
);

const MusicPlayer = () => {
  const [currentScreen, setCurrentScreen] = useState('home');
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [currentTrackIndex, setCurrentTrackIndex] = useState(0);
  const [volume, setVolume] = useState(0.5);
  const [isMuted, setIsMuted] = useState(false);
  const [audioError, setAudioError] = useState(false);
  const [isSeeking, setIsSeeking] = useState(false);
  const [isShuffled, setIsShuffled] = useState(false);
  const [isRepeating, setIsRepeating] = useState(false);
  const [shuffledIndices, setShuffledIndices] = useState([]);
  
  // Track data states
  const [dailySessionsTracks, setDailySessionsTracks] = useState([]);
  const [remakesTracks, setRemakesTracks] = useState([]);
  const [originalsTracks, setOriginalsTracks] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [fetchError, setFetchError] = useState(null);
  
  // Refs
  const playerRef = useRef(null);
  const rafRef = useRef(null);
  
  // Fetch tracks from Cloudinary
  useEffect(() => {
    const fetchTracks = async () => {
      try {
        setIsLoading(true);
        setFetchError(null);

        // Fetch from our Go server
        const response = await axios.get('https://music-meta.nskien.com/api/tracks');

        if (!response.data || !response.data.resources) {
          throw new Error('Invalid response from server');
        }

        // Process the resources into our track categories
        const resources = response.data.resources;
        
        // Sort resources into categories based on filename patterns
        const dailySessions = [];
        const remakes = [];
        const originals = [];

        resources.forEach(resource => {
          // Extract filename from public_id (remove folder prefix)
          const filename = resource.public_id.split('/').pop();
          
          // Create track object with basic Cloudinary URL
          const track = {
            id: resource.asset_id,
            filename,
            // Use simple direct URL
            audio: `https://res.cloudinary.com/${CLOUDINARY_CLOUD_NAME}/video/upload/${resource.public_id}.mp3`
          };

          // Add to appropriate category based on filename pattern
          if (filename.startsWith('cover-')) {
            Object.defineProperty(track, 'title', {
              get() { return formatRemakeTitle(this.filename); }
            });
            remakes.push(track);
          } else if (/^\d{4}-\d{2}-\d{2}/.test(filename)) {
            Object.defineProperty(track, 'date', {
              get() { return formatDateFromFilename(this.filename); }
            });
            dailySessions.push(track);
          } else {
            Object.defineProperty(track, 'title', {
              get() { return formatOriginalTitle(this.filename); }
            });
            originals.push(track);
          }
        });

        // Sort daily sessions by date (newest first)
        dailySessions.sort((a, b) => b.filename.localeCompare(a.filename));
        
        // Sort remakes and originals alphabetically by title
        remakes.sort((a, b) => a.title.localeCompare(b.title));
        originals.sort((a, b) => a.title.localeCompare(b.title));

        setDailySessionsTracks(dailySessions);
        setRemakesTracks(remakes);
        setOriginalsTracks(originals);

      } catch (error) {
        console.error('Error fetching tracks:', error);
        setFetchError(`Failed to fetch tracks: ${error.message}`);
      } finally {
        setIsLoading(false);
      }
    };
    
    fetchTracks();
  }, []);
  
  // Get current category tracks
  const getCurrentTracks = () => {
    switch(currentScreen) {
      case 'dailySessions':
        return dailySessionsTracks;
      case 'remakes':
        return remakesTracks;
      case 'originals':
        return originalsTracks;
      default:
        return dailySessionsTracks;
    }
  };
  
  const currentTracks = getCurrentTracks();
  const currentTrack = currentTracks[currentTrackIndex];
  
  // Track counts for the home screen
  const trackCounts = {
    dailySessions: dailySessionsTracks.length,
    remakes: remakesTracks.length,
    originals: originalsTracks.length
  };
  
  // Shuffle array utility function
  const shuffleArray = (array) => {
    const shuffled = [...array];
    for (let i = shuffled.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
    }
    return shuffled;
  };

  // Update shuffled indices when tracks change or shuffle is toggled
  useEffect(() => {
    if (isShuffled) {
      const tracks = getCurrentTracks();
      const indices = Array.from({ length: tracks.length }, (_, i) => i);
      setShuffledIndices(shuffleArray(indices));
    } else {
      setShuffledIndices([]);
    }
  }, [isShuffled, currentScreen, dailySessionsTracks, remakesTracks, originalsTracks]);

  const getNextTrackIndex = () => {
    const tracks = getCurrentTracks();
    if (isShuffled) {
      const currentShuffledIndex = shuffledIndices.indexOf(currentTrackIndex);
      if (currentShuffledIndex < shuffledIndices.length - 1) {
        return shuffledIndices[currentShuffledIndex + 1];
      }
      return -1;
    } else {
      return currentTrackIndex < tracks.length - 1 ? currentTrackIndex + 1 : -1;
    }
  };

  const getPrevTrackIndex = () => {
    if (isShuffled) {
      const currentShuffledIndex = shuffledIndices.indexOf(currentTrackIndex);
      if (currentShuffledIndex > 0) {
        return shuffledIndices[currentShuffledIndex - 1];
      }
      return -1;
    } else {
      return currentTrackIndex > 0 ? currentTrackIndex - 1 : -1;
    }
  };

  const handleNextTrack = () => {
    const nextIndex = getNextTrackIndex();
    if (nextIndex !== -1) {
      setCurrentTrackIndex(nextIndex);
      setIsPlaying(true);
    } else if (isRepeating) {
      // If repeating and at the end, go back to the first track
      setCurrentTrackIndex(isShuffled ? shuffledIndices[0] : 0);
      setIsPlaying(true);
    } else {
      setIsPlaying(false);
      setCurrentTime(0);
    }
  };

  const handlePrevTrack = () => {
    const prevIndex = getPrevTrackIndex();
    if (prevIndex !== -1) {
      setCurrentTrackIndex(prevIndex);
      setIsPlaying(true);
    }
  };

  const handleOnEnd = () => {
    if (isRepeating && !isShuffled) {
      // Single track repeat
      if (playerRef.current) {
        playerRef.current.seek(0);
        setIsPlaying(true);
      }
    } else {
      // Move to next track (handles both normal and shuffled playback)
      handleNextTrack();
    }
  };

  const toggleShuffle = () => {
    setIsShuffled(!isShuffled);
  };

  const toggleRepeat = () => {
    setIsRepeating(!isRepeating);
  };

  // Update seek position using requestAnimationFrame
  const updateSeekPosition = () => {
    if (playerRef.current && isPlaying && !isSeeking) {
      try {
        const seek = playerRef.current.seek();
        if (typeof seek === 'number' && !isNaN(seek)) {
          setCurrentTime(seek);
        }
      } catch (err) {
        console.error("Error during seek:", err);
      }
      rafRef.current = requestAnimationFrame(updateSeekPosition);
    }
  };

  // Initialize RAF when playing starts
  useEffect(() => {
    if (isPlaying && !isSeeking) {
      rafRef.current = requestAnimationFrame(updateSeekPosition);
    } else {
      cancelAnimationFrame(rafRef.current);
    }
    
    return () => cancelAnimationFrame(rafRef.current);
  }, [isPlaying, isSeeking]);

  // Reset state when track or category changes
  useEffect(() => {
    setCurrentTime(0);
    setDuration(0);
    setAudioError(false);
  }, [currentTrackIndex, currentScreen]);

  const handleSliderChange = (e) => {
    const newTime = parseFloat(e.target.value);
    setCurrentTime(newTime);
  };
  
  const handleSliderMouseDown = () => {
    setIsSeeking(true);
  };
  
  const handleSliderMouseUp = () => {
    if (playerRef.current) {
      try {
        playerRef.current.seek(currentTime);
      } catch (err) {
        console.error("Error during seek:", err);
      }
    }
    setIsSeeking(false);
  };

  const playTrack = (index) => {
    // If already playing this track, just toggle play/pause
    if (currentTrackIndex === index) {
      setIsPlaying(!isPlaying);
      return;
    }
    
    // Otherwise, change track and start playing
    setCurrentTrackIndex(index);
    setIsPlaying(true);
  };

  const togglePlayPause = () => {
    setIsPlaying(!isPlaying);
  };

  const handleOnLoad = () => {
    if (playerRef.current) {
      try {
        const duration = playerRef.current.duration();
        setDuration(duration);
        setAudioError(false);
      } catch (err) {
        console.error("Error during onLoad:", err);
      }
    }
  };

  const handleVolumeChange = (e) => {
    const newVolume = parseFloat(e.target.value);
    setVolume(newVolume);
    setIsMuted(newVolume === 0);
  };

  const toggleMute = () => {
    if (isMuted) {
      setIsMuted(false);
      setVolume(1.0);
    } else {
      setIsMuted(true);
      setVolume(0);
    }
  };

  const handleHowlerError = (id, err) => {
    console.error('Howler error:', err);
    setAudioError(true);
    setIsPlaying(false);
  };

  const handleNavigateBack = () => {
    setCurrentScreen('home');
    setIsPlaying(false);
  };

  const handleNavigate = (screen) => {
    setCurrentScreen(screen);
    setCurrentTrackIndex(0);
    setIsPlaying(false);
  };

  const playerControls = (
    <PlayerControls 
      isPlaying={isPlaying}
      onPlayPause={togglePlayPause}
      currentTime={currentTime}
      duration={duration}
      onSeek={handleSliderChange}
      onSeekStart={handleSliderMouseDown}
      onSeekEnd={handleSliderMouseUp}
      volume={volume}
      isMuted={isMuted}
      onVolumeChange={handleVolumeChange}
      onMuteToggle={toggleMute}
      isSeeking={isSeeking}
      currentTrack={currentTrack}
      isDailySession={currentScreen === 'dailySessions'}
      onPrevTrack={handlePrevTrack}
      onNextTrack={handleNextTrack}
      isShuffled={isShuffled}
      onShuffleToggle={toggleShuffle}
      isRepeating={isRepeating}
      onRepeatToggle={toggleRepeat}
    />
  );

  const getCategoryTitle = () => {
    switch(currentScreen) {
      case 'dailySessions':
        return 'Daily Sessions';
      case 'remakes':
        return 'Remakes';
      case 'originals':
        return 'Full Originals';
      default:
        return '';
    }
  };

  return (
    <div className="w-full max-w-6xl mx-auto h-screen max-h-[900px] border-4 border-gray-300 rounded-3xl overflow-hidden flex flex-col">
      {isLoading ? (
        <div className="flex items-center justify-center h-full">
          <p className="text-2xl">Loading tracks...</p>
        </div>
      ) : fetchError ? (
        <div className="flex items-center justify-center h-full">
          <div className="text-center">
            <p className="text-2xl text-red-500 mb-4">{fetchError}</p>
            <p>Using fallback tracks instead.</p>
          </div>
        </div>
      ) : currentScreen === 'home' ? (
        <HomeScreen onNavigate={handleNavigate} trackCounts={trackCounts} />
      ) : (
        <>
          <CategoryScreen 
            onNavigateBack={handleNavigateBack}
            tracks={currentTracks}
            currentTrackIndex={currentTrackIndex}
            isPlaying={isPlaying}
            onTrackSelect={playTrack}
            playerControls={playerControls}
            audioError={audioError}
            categoryTitle={getCategoryTitle()}
            isDailySession={currentScreen === 'dailySessions'}
            currentScreen={currentScreen}
          />
          {currentTrack && (
            <ReactHowler
              src={currentTrack.audio}
              playing={isPlaying}
              onLoad={handleOnLoad}
              onEnd={handleOnEnd}
              onLoadError={handleHowlerError}
              ref={playerRef}
              volume={volume}
              html5={true}
            />
          )}
        </>
      )}
    </div>
  );
};

const App = () => {
  return (
    <div className="min-h-screen w-full flex items-center justify-center p-0 md:p-4">
      <div className="w-full h-screen md:max-w-6xl mx-auto md:h-[700px] md:border-4 md:border-gray-300 md:rounded-3xl overflow-hidden flex flex-col bg-white">
        <MusicPlayer />
      </div>
    </div>
  );
};

export default App;
