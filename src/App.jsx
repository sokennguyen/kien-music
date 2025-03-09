import React, { useState, useRef, useEffect } from 'react';
import { Play, Pause, Volume2, ArrowLeft, VolumeX, Music, Headphones } from 'lucide-react';
import ReactHowler from 'react-howler';

// Helper function to format date from filename
const formatDateFromFilename = (audioUrl) => {
  // Extract the filename from the URL
  const filename = audioUrl.split('/').pop();
  
  // Extract the date part (assuming format: yyyy-mm-dd.mp3)
  const datePart = filename.split('.')[0];
  
  // Parse the date components
  const [year, month, day] = datePart.split('-').map(num => parseInt(num, 10));
  
  // Create a formatted date string
  const date = new Date(year, month - 1, day); // month is 0-indexed in JS Date
  
  // Format options
  const options = { year: 'numeric', month: 'long', day: 'numeric' };
  return date.toLocaleDateString('en-US', options);
};

// Helper function to format remake title from filename
const formatRemakeTitle = (audioUrl) => {
  try {
    // Extract the filename from the URL
    const filename = audioUrl.split('/').pop();
    
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
const formatOriginalTitle = (audioUrl) => {
  try {
    // Extract the filename from the URL
    const filename = audioUrl.split('/').pop();
    
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

// Daily Sessions tracks - using dates extracted from filenames
const dailySessionsTracks = [
  { 
    id: 1, 
    audio: "https://res.cloudinary.com/drenighdk/video/upload/v1741270079/my-music/2025-03-01.mp3",
    get date() { return formatDateFromFilename(this.audio); }
  },
  { 
    id: 2, 
    audio: "https://res.cloudinary.com/drenighdk/video/upload/v1741270079/my-music/2024-11-01.mp3",
    get date() { return formatDateFromFilename(this.audio); }
  },
  { 
    id: 3, 
    audio: "https://res.cloudinary.com/drenighdk/video/upload/v1741270079/my-music/2025-06-03.mp3",
    get date() { return formatDateFromFilename(this.audio); }
  },
  { 
    id: 4, 
    audio: "https://res.cloudinary.com/drenighdk/video/upload/v1741270078/my-music/2025-02-18.mp3",
    get date() { return formatDateFromFilename(this.audio); }
  },
];

// Remakes tracks
const remakesTracks = [
  { 
    id: 1, 
    audio: "https://res.cloudinary.com/drenighdk/video/upload/v1741270159/my-music/cover-Clyde___Bonnie-Tai_Verdes.mp3",
    get title() { return formatRemakeTitle(this.audio); }
  },
  { 
    id: 2, 
    audio: "https://res.cloudinary.com/drenighdk/video/upload/v1741270159/my-music/cover-Simon___Garfunkel-The_Sound_of_Silence.mp3",
    get title() { return formatRemakeTitle(this.audio); }
  },
];

// Full Originals tracks
const originalsTracks = [
  // Empty for now
];

const formatTime = (timeInSeconds) => {
  if (!timeInSeconds || isNaN(timeInSeconds)) return "00:00";
  const minutes = Math.floor(timeInSeconds / 60);
  const seconds = Math.floor(timeInSeconds % 60);
  return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
};

const HomeScreen = ({ onNavigate }) => (
  <div className="flex flex-col h-full p-8 font-sans">
    <div className="flex justify-between items-center mb-16">
      <div className="text-2xl flex items-center">
        <a 
          href="#spotify" 
          className="mr-8 text-gray-700 hover:text-green-500 transition-colors duration-300"
          title="Spotify"
        >
          <Music className="w-8 h-8" />
        </a>
        <a 
          href="#soundcloud" 
          className="text-gray-700 hover:text-orange-500 transition-colors duration-300"
          title="SoundCloud"
        >
          <Headphones className="w-8 h-8" />
        </a>
      </div>
      <h1 className="text-4xl font-bold">Kien's Homemade Music</h1>
    </div>
    
    <div className="grid grid-cols-3 gap-8">
      <div 
        className="flex flex-col items-start cursor-pointer" 
        onClick={() => onNavigate('dailySessions')}
      >
        <div className="w-64 h-64 rounded-lg border-4 border-gray-300 mb-4"></div>
        <h2 className="text-2xl font-bold text-left">Daily Sessions</h2>
        <p className="text-lg text-left">{dailySessionsTracks.length} Tracks</p>
      </div>
      
      <div 
        className="flex flex-col items-start cursor-pointer"
        onClick={() => onNavigate('remakes')}
      >
        <div className="w-64 h-64 rounded-lg border-4 border-gray-300 mb-4"></div>
        <h2 className="text-2xl font-bold text-left">Remakes</h2>
        <p className="text-lg text-left">{remakesTracks.length} Tracks</p>
      </div>
      
      <div 
        className="flex flex-col items-start cursor-pointer"
        onClick={() => onNavigate('originals')}
      >
        <div className="w-64 h-64 rounded-lg border-4 border-gray-300 mb-4"></div>
        <h2 className="text-2xl font-bold text-left">Full Originals</h2>
        <p className="text-lg text-left">{originalsTracks.length > 0 ? `${originalsTracks.length} Tracks` : "Work in progress, just wait"}</p>
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
  currentTrack
}) => (
  <div className="p-8 flex flex-col border-t border-gray-300 bg-white">
    {currentTrack && (
      <div className="text-left mb-4 w-full">
        <h3 className="text-2xl font-bold text-blue-600">
          {currentTrack.title || currentTrack.date}
        </h3>
      </div>
    )}
    <div className="flex items-center justify-center w-full">
      <button 
        onClick={onPlayPause} 
        className="mr-8 flex-shrink-0 cursor-pointer hover:opacity-80 active:opacity-60"
      >
        {isPlaying ? (
          <Pause className="w-16 h-16 text-blue-500" />
        ) : (
          <Play className="w-16 h-16 text-blue-500" />
        )}
      </button>
      
      <span className="mr-4 w-24 text-right flex-shrink-0 text-xl">{formatTime(currentTime)}</span>
      
      <div className="flex-grow mx-8">
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
          className={`w-full cursor-pointer h-3 ${isSeeking ? 'seeking' : ''}`}
          step="0.1"
        />
      </div>
      
      <span className="w-24 flex-shrink-0 text-xl">{formatTime(duration)}</span>
      
      <div className="flex items-center ml-4">
        <button 
          onClick={onMuteToggle} 
          className="flex-shrink-0 mr-4 cursor-pointer hover:opacity-80 active:opacity-60"
        >
          {isMuted ? (
            <VolumeX className="w-12 h-12" />
          ) : (
            <Volume2 className="w-12 h-12" />
          )}
        </button>
        <input 
          type="range"
          min="0"
          max="1"
          step="0.01"
          value={volume}
          onChange={onVolumeChange}
          className="w-32 cursor-pointer h-3"
        />
      </div>
    </div>
  </div>
);

const TrackList = ({ tracks, currentTrackIndex, isPlaying, onTrackSelect, isDailySession }) => (
  <div className="w-2/3 flex flex-col overflow-y-auto">
    {tracks.map((track, index) => (
      <div 
        key={track.id} 
        className={`border-b border-gray-300 py-8 px-8 flex justify-between items-center cursor-pointer hover:bg-blue-100 ${currentTrackIndex === index ? 'bg-blue-50' : ''}`}
        onClick={() => onTrackSelect(index)}
      >
        <div className="flex-grow text-left">
          {isDailySession ? (
            <p className="text-xl font-medium text-left">{track.date}</p>
          ) : (
            <p className="text-xl font-medium text-left">{track.title}</p>
          )}
        </div>
        <div className="ml-4">
          {currentTrackIndex === index && isPlaying ? (
            <Pause className="w-10 h-10 text-blue-500" />
          ) : (
            <Play className="w-10 h-10" />
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
  isDailySession = false
}) => (
  <div className="flex flex-col h-full">
    <div className="flex flex-1 overflow-hidden">
      <div className="w-1/3 p-8 border-r border-gray-300 flex flex-col items-center">
        <button 
          className="mb-8 flex items-center text-blue-500 cursor-pointer hover:opacity-80 active:opacity-60 self-start" 
          onClick={onNavigateBack}
        >
          <ArrowLeft className="w-8 h-8 mr-2" />
          <span className="text-xl">Back</span>
        </button>
        <div className="w-full aspect-square rounded-lg border-4 border-gray-300 mb-8"></div>
        <h2 className="text-3xl font-bold text-center">{categoryTitle}</h2>
        <p className="text-xl text-center">{tracks.length} Tracks</p>
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
        <div className="p-4 text-xl text-red-500 text-center border-t border-red-200 bg-red-50">
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
  const [volume, setVolume] = useState(1.0);
  const [isMuted, setIsMuted] = useState(false);
  const [audioError, setAudioError] = useState(false);
  const [isSeeking, setIsSeeking] = useState(false);
  
  // Refs
  const playerRef = useRef(null);
  const rafRef = useRef(null);
  
  // Get current category tracks based on screen
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

  // Clean up on unmount
  useEffect(() => {
    return () => {
      cancelAnimationFrame(rafRef.current);
      if (playerRef.current) {
        playerRef.current.unload();
      }
    };
  }, []);

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

  const handleOnEnd = () => {
    if (currentTrackIndex < currentTracks.length - 1) {
      setCurrentTrackIndex(currentTrackIndex + 1);
      // Keep playing
      setIsPlaying(true);
    } else {
      setIsPlaying(false);
      setCurrentTime(0);
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
      {currentScreen === 'home' ? (
        <HomeScreen onNavigate={handleNavigate} />
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

// App wrapper to ensure proper centering
const App = () => {
  return (
    <div className="min-h-screen w-full flex items-center justify-center">
      <MusicPlayer />
    </div>
  );
};

export default App;
